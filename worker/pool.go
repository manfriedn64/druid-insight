package worker

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"time"

	"druid-insight/druid"
	"druid-insight/logging"
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
func StartReportWorkers(num int, druidCfg *druid.DruidConfig, reportLogger *logging.Logger) {
	for i := 0; i < num; i++ {
		go reportWorker(druidCfg, reportLogger)
	}
}

// Un worker traite une requête à la fois, dès qu’il en trouve une dans la file FIFO
func reportWorker(druidCfg *druid.DruidConfig, reportLogger *logging.Logger) {
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
		processingRequests.Store(nextID, &ReportResult{Status: StatusProcessing})

		reportLogger.Write("[START] id=" + nextID + " owner=" + req.Owner)

		status, result, csvPath, errMsg := ProcessRequest(req, druidCfg, reportLogger)
		processingRequests.Store(nextID, &ReportResult{
			Status:   status,
			Result:   result,
			CSVPath:  csvPath,
			ErrorMsg: errMsg,
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
func ProcessRequest(req *ReportRequest, druidCfg *druid.DruidConfig, logger *logging.Logger) (ReportStatus, interface{}, string, string) {
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
					return StatusError, nil, "", "Intervalle invalide"
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
		return StatusError, nil, "", "Datasource inconnue"
	}
	drFilters := druid.ConvertFiltersToDruidDimFilter(filters, ds)

	granularity := "all"
	if tg, ok := req.Payload["time_group"].(string); ok && tg != "" {
		// Druid supporte hour, day, week, month (par défaut en minuscule)
		granularity = tg
	}
	// 2. Construire la requête groupBy via BuildDruidQuery
	query, err := druid.BuildDruidQuery(
		req.Datasource,
		dims,
		mets,
		drFilters,
		intervals,
		ds,
		granularity,
	)
	if err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s buildquery: %v", req.ID, err))
		return StatusError, nil, "", "Erreur construction requête Druid"
	}

	// 3. Exécuter la requête avec ExecuteDruidQuery
	results, err := druid.ExecuteDruidQuery(druidCfg.HostURL+"/druid/v2/", query)
	if err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s druid error: %v", req.ID, err))
		return StatusError, nil, "", fmt.Sprintf("Erreur Druid: %v", err)
	}

	// 4. Générer un CSV dans csv/<id>.csv
	csvDir := "csv"
	if err := os.MkdirAll(csvDir, 0755); err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s mkdir csv: %v", req.ID, err))
		return StatusError, nil, "", "Impossible de créer le dossier csv/"
	}
	csvPath := filepath.Join(csvDir, req.ID+".csv")

	f, err := os.Create(csvPath)
	if err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s create csv: %v", req.ID, err))
		return StatusError, nil, "", "Impossible de créer le fichier CSV"
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if len(results) == 0 {
		// Tente quand même d’extraire les headers pour un CSV vide
		var headers []string
		// Optionnel : tu peux garder les headers précédemment calculés, ou tenter une structure attendue
		// Si pas de headers connus, log + continue quand même
		// Écrit le CSV vide (headers seuls)
		if len(headers) > 0 {
			if err := w.Write(headers); err != nil {
				logger.Write(fmt.Sprintf("[FAIL] id=%s write csv header: %v", req.ID, err))
				return StatusError, nil, "", "Erreur d'écriture CSV"
			}
		}
		logger.Write(fmt.Sprintf("[COMPLETE] id=%s aucun résultat (fichier CSV vide)", req.ID))
		return StatusComplete, results, csvPath, "Aucune donnée retournée par Druid"
	}

	// La structure de résultat Druid groupBy = [{event: {...}}, ...]
	var headers []string
	for _, res := range results {
		if evt, ok := res["event"].(map[string]interface{}); ok && len(evt) > 0 {
			for k := range evt {
				headers = append(headers, k)
			}
			break
		}
	}
	if len(headers) == 0 {
		logger.Write(fmt.Sprintf("[FAIL] id=%s pas d'entêtes", req.ID))
		return StatusError, nil, "", "Impossible d'extraire les colonnes"
	}
	if err := w.Write(headers); err != nil {
		logger.Write(fmt.Sprintf("[FAIL] id=%s write csv header: %v", req.ID, err))
		return StatusError, nil, "", "Erreur d'écriture CSV"
	}

	// Ecriture des lignes
	for _, res := range results {
		var rec []string
		if evt, ok := res["event"].(map[string]interface{}); ok {
			for _, col := range headers {
				val := evt[col]
				if col == "time" && val != nil {
					// Conversion timestamp -> string
					// Le timestamp Druid est en millisecondes
					var t time.Time
					switch val := val.(type) {
					case float64:
						t = time.Unix(int64(val)/1000, (int64(val)%1000)*int64(time.Millisecond))
					case int64:
						t = time.Unix(val/1000, (val%1000)*int64(time.Millisecond))
					case string:
						i, err := strconv.Atoi(val)
						if err != nil {
							// Peut arriver si Druid est en mode ISO string
							parsed, err := time.Parse(time.RFC3339, val)
							if err == nil {
								t = parsed
							} else {
								// valeur non parseable, tu la laisses brute
								rec = append(rec, fmt.Sprintf("%s", val))
								continue
							}
						} else {
							t = time.Unix(int64(i)/1000, (int64(i)%1000)*int64(time.Millisecond))
						}
					default:
						// type inconnu, laisses brute
						rec = append(rec, fmt.Sprintf("%s", val))
						continue
					}
					// Format voulu :
					s := t.Format("2006-01-02 15") // YYYY-MM-DD HH
					rec = append(rec, s)
				} else {
					switch v := val.(type) {
					case int, int8, int16, int32, int64:
						rec = append(rec, fmt.Sprintf("%d", v))
					case uint, uint8, uint16, uint32, uint64:
						rec = append(rec, fmt.Sprintf("%d", v))
					case float64:
						// Pour les floats, sans notation scientifique, entier si c'est censé l'être
						rec = append(rec, strconv.FormatFloat(v, 'f', -1, 64))
					case float32:
						rec = append(rec, strconv.FormatFloat(float64(v), 'f', -1, 32))
					case string:
						rec = append(rec, v)
					case nil:
						rec = append(rec, "")
					default:
						// fallback : affichage brut (rare)
						// Pour les types numériques encodés dynamiquement en float64 (ex : Druid renvoie souvent int en float64)
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
				return StatusError, nil, "", "Erreur d'écriture CSV"
			}
		}
	}

	logger.Write(fmt.Sprintf("[COMPLETE] id=%s lignes=%d fichier=%s", req.ID, len(results), csvPath))
	return StatusComplete, results, csvPath, ""
}
