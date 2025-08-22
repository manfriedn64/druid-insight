package worker

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"time"

	"druid-insight/auth"
	"druid-insight/config"
	"druid-insight/druid"
	"druid-insight/logging"

	"github.com/tealeg/xlsx/v3"
)

// Maps et file d’attente FIFO
var (
	pendingRequests    = sync.Map{} // id => *ReportRequest
	processingRequests = sync.Map{} // id => *ReportResult
	pendingMutex       = &sync.Mutex{}
	pendingOrder       = []string{}
)

// Ajoute une requête dans la file FIFO
func AddPendingRequest(req *ReportRequest) {
	pendingRequests.Store(req.ID, req)
	pendingMutex.Lock()
	pendingOrder = append(pendingOrder, req.ID)
	pendingMutex.Unlock()
}

// Récupère puis supprime la plus ancienne requête FIFO (ou "" si aucune)
func NextPendingID() string {
	pendingMutex.Lock()
	defer pendingMutex.Unlock()
	if len(pendingOrder) == 0 {
		return ""
	}
	nextID := pendingOrder[0]
	pendingOrder = pendingOrder[1:]
	return nextID
}

// Expose les maps pour l’API statut
func PendingRequests() *sync.Map    { return &pendingRequests }
func ProcessingRequests() *sync.Map { return &processingRequests }

// Lance N workers en parallèle
func StartReportWorkers(num int, druidCfg *config.DruidConfig, reportLogger *logging.Logger, cfg *auth.Config) {
	for i := 0; i < num; i++ {
		go reportWorker(druidCfg, reportLogger, cfg)
	}
}

// Un worker traite une requête à la fois, dès qu’il en trouve une dans la file FIFO
func reportWorker(druidCfg *config.DruidConfig, reportLogger *logging.Logger, cfg *auth.Config) {
	for {
		nextID := NextPendingID()
		if nextID == "" {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		v, ok := pendingRequests.LoadAndDelete(nextID)
		if !ok {
			continue
		}
		req := v.(*ReportRequest)
		processingRequests.Store(nextID, &ReportResult{
			Status: StatusProcessing,
			Owner:  req.Owner, // <-- Ajout ici
		})

		reportLogger.Write("[START] id=" + nextID + " owner=" + req.Owner)

		status, result, csvPath, xlsPath, errMsg := ProcessRequest(req, druidCfg, reportLogger, cfg)
		processingRequests.Store(nextID, &ReportResult{
			Status:   status,
			Result:   result,
			CSVPath:  csvPath,
			XLSPath:  xlsPath,
			ErrorMsg: errMsg,
			Owner:    req.Owner, // <-- Ajout ici aussi
		})
	}
}

func ComputeIntervals(start, end, compare string) (mainInterval, compareInterval string, err error) {
	const layoutInput = "2006-01-02"
	const layoutOutput = "2006-01-02T15:04:05Z"

	startT, err := time.Parse(layoutInput, start)
	if err != nil {
		return "", "", err
	}
	endT, err := time.Parse(layoutInput, end)
	if err != nil {
		return "", "", err
	}
	// Pour couvrir toute la journée end incluse, on rajoute 1 jour à endT (convention Druid "end exclusive")
	endT = endT.AddDate(0, 0, 1)

	mainInterval = startT.Format(layoutOutput) + "/" + endT.Format(layoutOutput)
	periodDuration := endT.Sub(startT)

	var compareStart, compareEnd time.Time

	switch compare {
	case "prev_day":
		compareEnd = startT
		compareStart = compareEnd.Add(-periodDuration)
	case "prev_week":
		if periodDuration > 7*time.Hour*24 {
			compareEnd = startT.AddDate(0, 0, -7)
			compareStart = compareEnd.Add(-periodDuration)
		} else {
			compareStart = startT.AddDate(0, 0, -7)
			compareEnd = endT.AddDate(0, 0, -7)
		}
	case "prev_month":
		if periodDuration > 28*time.Hour*24 {
			compareEnd = startT.AddDate(0, -1, 0)
			compareStart = compareEnd.Add(-periodDuration)
		} else {
			compareStart = startT.AddDate(0, -1, 0)
			compareEnd = endT.AddDate(0, -1, 0)
		}
	case "prev_year":
		if periodDuration > 365*time.Hour*24 {
			compareEnd = startT.AddDate(-1, 0, 0)
			compareStart = compareEnd.Add(-periodDuration)
		} else {
			compareStart = startT.AddDate(-1, 0, 0)
			compareEnd = endT.AddDate(-1, 0, 0)
		}
	default:
		return mainInterval, "", nil
	}

	compareInterval = compareStart.Format(layoutOutput) + "/" + compareEnd.Format(layoutOutput)
	return mainInterval, compareInterval, nil
}

// Utilise les helpers du module druid pour exécuter la requête et générer un CSV
func ProcessRequest(req *ReportRequest, druidCfg *config.DruidConfig, logger *logging.Logger, cfg *auth.Config) (ReportStatus, interface{}, string, string, string) {
	// Récupération des paramètres attendus dans le payload (dimensions, metrics, filters, intervals)
	var dims, mets []string
	var filters []interface{}
	var intervals []string

	// Parsing souple pour extraire les params du payload
	if v, ok := req.Payload["dimensions"]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, d := range arr {
				if s, ok := d.(string); ok {
					dims = append(dims, s)
				}
			}
		}
	}
	if v, ok := req.Payload["metrics"]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, m := range arr {
				if s, ok := m.(string); ok {
					mets = append(mets, s)
				}
			}
		}
	}
	if v, ok := req.Payload["filters"]; ok {
		if arr, ok := v.([]interface{}); ok {
			filters = arr
		}
	}
	if v, ok := req.Payload["dates"]; ok {
		if arr, ok := v.([]interface{}); ok && len(arr) == 2 {
			start, ok1 := arr[0].(string)
			end, ok2 := arr[1].(string)
			if ok1 && ok2 {
				compare := ""
				if c, ok := req.Payload["compare"].(string); ok {
					compare = c
				}
				mainInterval, compareInterval, err := ComputeIntervals(start, end, compare)
				if err != nil {
					logger.Write(fmt.Sprintf("[FAIL] id=%s bad interval: %v", req.ID, err))
					return StatusError, nil, "", "", "Intervalle invalide"
				}
				intervals = append(intervals, mainInterval)
				if compareInterval != "" {
					intervals = append(intervals, compareInterval)
				}
			}
		}
	}

	// 1. Retrouver la config de la datasource
	ds, ok := druidCfg.Datasources[req.Datasource]
	if !ok {
		logger.Write(fmt.Sprintf("[FAIL] id=%s unknown datasource %s", req.ID, req.Datasource))
		return StatusError, nil, "", "", "Datasource inconnue"
	}

	granularity := "all"
	if tg, ok := req.Payload["time_group"].(string); ok && tg != "" {
		// Druid supporte hour, day, week, month (par défaut en minuscule)
		granularity = tg
	}

	query, err := druid.BuildDruidQuery(
		req.Datasource,
		dims,
		mets,
		filters, // les filtres utilisateur bruts
		intervals,
		ds,
		granularity,
		req.Owner,
		req.Admin,
		druidCfg,
		cfg,
		req.Owner,
		req.Context,
	)
	if err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s buildquery: %v", req.ID, err))
		return StatusError, nil, "", "", "Erreur construction requête Druid"
	}

	// 3. Exécuter la requête avec ExecuteDruidQuery
	results, err := druid.ExecuteDruidQuery(druidCfg.HostURL+"/druid/v2/", query)
	if err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s druid error: %v", req.ID, err))
		return StatusError, nil, "", "", fmt.Sprintf("Erreur Druid: %v", err)
	}

	// 4. Générer un CSV dans csv/<id>.csv
	csvDir := filepath.Join("reports", req.Owner, "csv")
	xlsDir := filepath.Join("reports", req.Owner, "xls")

	if err := os.MkdirAll(csvDir, 0755); err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s mkdir csv: %v", req.ID, err))
		return StatusError, nil, "", "", "Impossible de créer le dossier csv/"
	}
	csvPath := filepath.Join(csvDir, req.ID+".csv")
	xlsPath := filepath.Join(xlsDir, req.ID+".xlsx")

	f, err := os.Create(csvPath)
	if err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s create csv: %v", req.ID, err))
		return StatusError, nil, "", "", "Impossible de créer le fichier CSV"
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Construction des headers triés ---
	// On veut: time (si présent), puis dimensions (alpha), puis métriques (alpha)
	var headers []string
	// Récupère les dimensions et métriques demandées
	dimSet := map[string]bool{}
	for _, d := range dims {
		dimSet[d] = true
	}
	metSet := map[string]bool{}
	for _, m := range mets {
		metSet[m] = true
	}
	// Détecte la colonne time
	hasTime := dimSet["time"]
	// Trie les dimensions et métriques (hors time)
	var dimHeaders, metHeaders []string
	for d := range dimSet {
		if d != "time" {
			dimHeaders = append(dimHeaders, d)
		}
	}
	for m := range metSet {
		metHeaders = append(metHeaders, m)
	}
	sort.Strings(dimHeaders)
	sort.Strings(metHeaders)
	if hasTime {
		headers = append(headers, "time")
	}
	headers = append(headers, dimHeaders...)
	headers = append(headers, metHeaders...)

	if len(results) == 0 {
		// Écrit le CSV vide (headers seuls)
		if len(headers) > 0 {
			if err := w.Write(headers); err != nil {
				logger.Write(fmt.Sprintf("[FAIL] id=%s write csv header: %v", req.ID, err))
				return StatusError, nil, "", "", "Erreur d'écriture CSV"
			}
		}
		logger.Write(fmt.Sprintf("[COMPLETE] id=%s aucun résultat (fichier CSV vide)", req.ID))
		return StatusComplete, results, csvPath, "", "Aucune donnée retournée par Druid"
	}

	// Utilise les headers triés pour écrire le CSV ---
	if err := w.Write(headers); err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s write csv header: %v", req.ID, err))
		return StatusError, nil, "", "", "Erreur d'écriture CSV"
	}

	// Ecriture des lignes
	for _, res := range results {
		var rec []string
		if evt, ok := res["event"].(map[string]interface{}); ok {
			for _, col := range headers {
				val := evt[col]
				if col == "time" && val != nil {
					// Conversion timestamp -> string
					var t time.Time
					switch val := val.(type) {
					case float64:
						t = time.Unix(int64(val)/1000, (int64(val)%1000)*int64(time.Millisecond))
					case int64:
						t = time.Unix(val/1000, (val%1000)*int64(time.Millisecond))
					case string:
						i, err := strconv.Atoi(val)
						if err != nil {
							parsed, err := time.Parse(time.RFC3339, val)
							if err == nil {
								t = parsed
							} else {
								rec = append(rec, fmt.Sprintf("%s", val))
								continue
							}
						} else {
							t = time.Unix(int64(i)/1000, (int64(i)%1000)*int64(time.Millisecond))
						}
					default:
						rec = append(rec, fmt.Sprintf("%s", val))
						continue
					}
					s := t.Format("2006-01-02 15")
					rec = append(rec, s)
				} else {
					switch v := val.(type) {
					case int, int8, int16, int32, int64:
						rec = append(rec, fmt.Sprintf("%d", v))
					case uint, uint8, uint16, uint32, uint64:
						rec = append(rec, fmt.Sprintf("%d", v))
					case float64:
						rec = append(rec, strconv.FormatFloat(v, 'f', -1, 64))
					case float32:
						rec = append(rec, strconv.FormatFloat(float64(v), 'f', -1, 32))
					case string:
						rec = append(rec, v)
					case nil:
						rec = append(rec, "")
					default:
						rv := reflect.ValueOf(v)
						if rv.Kind() == reflect.Float64 {
							rec = append(rec, strconv.FormatFloat(rv.Float(), 'f', -1, 64))
						} else if rv.Kind() == reflect.Int64 || rv.Kind() == reflect.Int {
							rec = append(rec, fmt.Sprintf("%d", rv.Int()))
						} else {
							rec = append(rec, fmt.Sprintf("%v", v))
						}
					}
				}
			}
			if err := w.Write(rec); err != nil {
				logger.Write(fmt.Sprintf("[FAIL] id=%s write csv row: %v", req.ID, err))
				return StatusError, nil, "", "", "Erreur d'écriture CSV"
			}
		}
	}

	logger.Write(fmt.Sprintf("[COMPLETE] id=%s lignes=%d fichier=%s", req.ID, len(results), csvPath))

	// Génération du fichier Excel
	if err := os.MkdirAll(xlsDir, 0755); err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s mkdir xls: %v", req.ID, err))
		return StatusError, nil, "", "", "Impossible de créer le dossier xls/"
	}

	xlsxFile := xlsx.NewFile()
	sheet, err := xlsxFile.AddSheet("Report")
	if err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s create xlsx sheet: %v", req.ID, err))
	} else {
		// Ecriture de l'en-tête
		row := sheet.AddRow()
		for _, h := range headers {
			row.AddCell().SetString(h)
		}

		// Préparation pour le typage des colonnes
		colIsFloat := make(map[int]bool)
		colIsInt := make(map[int]bool)
		colValues := make([][]interface{}, len(headers))

		// Collecte des valeurs pour typage
		for _, res := range results {
			if evt, ok := res["event"].(map[string]interface{}); ok {
				for i, col := range headers {
					val := evt[col]
					colValues[i] = append(colValues[i], val)
				}
			}
		}
		// Détection du type de chaque colonne
		for i, vals := range colValues {
			isInt, isFloat := true, true
			for _, v := range vals {
				switch vv := v.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
					// ok
				case float64, float32:
					f, _ := anyToFloat64(vv)
					if f != float64(int64(f)) {
						isInt = false
					}
				case string:
					// Tente de parser comme int
					if n, err := strconv.ParseInt(vv, 10, 64); err == nil {
						if float64(n) != float64(int64(float64(n))) {
							isInt = false
						}
						continue
					}
					// Tente de parser comme float
					if f, err := strconv.ParseFloat(vv, 64); err == nil {
						if f != float64(int64(f)) {
							isInt = false
						}
						continue
					}
					// Sinon, ce n'est pas un nombre
					isInt = false
					isFloat = false
					break
				default:
					isInt = false
					isFloat = false
					break
				}
			}
			if isInt {
				colIsInt[i] = true
				colIsFloat[i] = false
			} else if isFloat {
				colIsInt[i] = false
				colIsFloat[i] = true
			} else {
				colIsInt[i] = false
				colIsFloat[i] = false
			}
		}

		// Ecriture des lignes
		for _, res := range results {
			if evt, ok := res["event"].(map[string]interface{}); ok {
				row := sheet.AddRow()
				for i, col := range headers {
					val := evt[col]
					cell := row.AddCell()
					if col == "time" && val != nil {
						var t time.Time
						switch val := val.(type) {
						case float64:
							t = time.Unix(int64(val)/1000, (int64(val)%1000)*int64(time.Millisecond))
						case int64:
							t = time.Unix(val/1000, (val%1000)*int64(time.Millisecond))
						case string:
							i, err := strconv.Atoi(val)
							if err != nil {
								parsed, err := time.Parse(time.RFC3339, val)
								if err == nil {
									t = parsed
								} else {
									cell.SetString(val)
									continue
								}
							} else {
								t = time.Unix(int64(i)/1000, (int64(i)%1000)*int64(time.Millisecond))
							}
						default:
							cell.SetString(fmt.Sprintf("%v", val))
							continue
						}
						cell.SetString(t.Format("2006-01-02 15"))
					} else if colIsFloat[i] {
						// Affichage arrondi à 2 décimales
						switch v := val.(type) {
						case float64:
							cell.SetFloatWithFormat(v, "0.00")
						case float32:
							cell.SetFloatWithFormat(float64(v), "0.00")
						case string:
							f, err := strconv.ParseFloat(v, 64)
							if err == nil {
								cell.SetFloatWithFormat(f, "0.00")
							} else {
								cell.SetString(v)
							}
						default:
							cell.SetString(fmt.Sprintf("%v", v))
						}
					} else if colIsInt[i] {
						// Écriture en tant qu'entier natif
						switch v := val.(type) {
						case int, int8, int16, int32, int64:
							cell.SetInt64(reflect.ValueOf(v).Int())
						case uint, uint8, uint16, uint32, uint64:
							cell.SetInt64(int64(reflect.ValueOf(v).Uint()))
						case float64, float32:
							f, _ := anyToFloat64(v)
							cell.SetInt64(int64(f))
						case string:
							if n, err := strconv.ParseInt(v, 10, 64); err == nil {
								cell.SetInt64(n)
							} else if f, err := strconv.ParseFloat(v, 64); err == nil {
								cell.SetInt64(int64(f))
							} else {
								cell.SetString("")
							}
						default:
							cell.SetString("")
						}
					} else {
						// Texte
						if v, ok := val.(string); ok {
							cell.SetString(v)
						} else {
							cell.SetString(fmt.Sprintf("%v", val))
						}
					}
				}
			}
		}
		// Sauvegarde du fichier
		if err := xlsxFile.Save(xlsPath); err != nil {
			logger.Write(fmt.Sprintf("[FAIL] id=%s write xlsx: %v", req.ID, err))
		}
	}

	return StatusComplete, results, csvPath, xlsPath, ""
}

func anyToFloat64(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case int32:
		return float64(x), true
	case int16:
		return float64(x), true
	case int8:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint64:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint8:
		return float64(x), true
	default:
		return 0, false
	}
}
