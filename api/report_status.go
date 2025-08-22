package api

import (
	"druid-insight/auth"
	"druid-insight/worker"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func ReportStatusHandler(cfg *auth.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _, err := auth.ExtractUserAndAdminFromJWT(r, cfg.JWT.Secret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing id", http.StatusBadRequest)
			return
		}
		maxAge := time.Duration(cfg.MaxFileAgeHours) * time.Hour
		// 1. Statut en mémoire
		if _, ok := worker.PendingRequests().Load(id); ok {
			json.NewEncoder(w).Encode(map[string]string{
				"status": string(worker.StatusWaiting),
			})
			return
		}
		if val, ok := worker.ProcessingRequests().Load(id); ok {
			rr := val.(*worker.ReportResult)
			// Vérifie que l'utilisateur est bien le propriétaire
			origReqVal, ok := worker.PendingRequests().Load(id)
			if !ok {
				origReqVal, ok = worker.ProcessingRequests().Load(id)
			}
			if ok {
				origReq := origReqVal.(*worker.ReportResult)
				if origReq.Owner != username {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			} else if val, ok := worker.ProcessingRequests().Load(id); ok {
				origRes := val.(*worker.ReportResult)
				if origRes.Owner != username { // Ajoute Owner dans ReportResult si ce n'est pas déjà fait
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			out := map[string]interface{}{
				"status": rr.Status,
			}
			if rr.Status == worker.StatusComplete {
				// Vérification d'expiration sur le fichier CSV ou XLS
				var filePath string
				if rr.CSVPath != "" {
					filePath = rr.CSVPath
				} else if rr.XLSPath != "" {
					filePath = rr.XLSPath
				}
				if filePath != "" {
					if fi, err := os.Stat(filePath); err == nil {
						age := time.Since(fi.ModTime())
						if maxAge > 0 && age > maxAge {
							json.NewEncoder(w).Encode(map[string]string{
								"status": string(worker.StatusExpired),
							})
							return
						}
					}
				}
				out["csv"] = rr.CSVPath
				out["excel"] = rr.XLSPath
			}
			if rr.Status == worker.StatusError {
				out["error"] = rr.ErrorMsg
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		// 2. Fichier existant mais id inconnu en mémoire
		csvPath := filepath.Join("csv", id+".csv")
		xlsPath := filepath.Join("xls", id+".xlsx")
		var filePath string
		if _, err := os.Stat(csvPath); err == nil {
			filePath = csvPath
		} else if _, err := os.Stat(xlsPath); err == nil {
			filePath = xlsPath
		}
		if filePath != "" {
			fi, err := os.Stat(filePath)
			if err == nil {
				age := time.Since(fi.ModTime())
				if maxAge > 0 && age > maxAge {
					json.NewEncoder(w).Encode(map[string]string{
						"status": string(worker.StatusExpired),
					})
					return
				}
				// Fichier trouvé et pas expiré
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": string(worker.StatusComplete),
					"csv":    csvPath,
					"excel":  xlsPath,
				})
				return
			}
		}
		// 3. Statut inconnu
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unknown",
		})
	}
}
