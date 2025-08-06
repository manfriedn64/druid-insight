package api

import (
	"druid-insight/auth"
	"druid-insight/worker"
	"encoding/json"
	"net/http"
)

func ReportStatusHandler(cfg *auth.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, err := auth.ExtractUserAndAdminFromJWT(r, cfg.JWT.Secret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing id", http.StatusBadRequest)
			return
		}
		if _, ok := worker.PendingRequests().Load(id); ok {
			json.NewEncoder(w).Encode(map[string]string{
				"status": string(worker.StatusWaiting),
			})
			return
		}
		if val, ok := worker.ProcessingRequests().Load(id); ok {
			rr := val.(*worker.ReportResult)
			out := map[string]interface{}{
				"status": rr.Status,
			}
			/*if rr.Status == worker.StatusComplete {
				out["result"] = rr.Result
				out["csv"] = rr.CSVPath
			}*/
			if rr.Status == worker.StatusError {
				out["error"] = rr.ErrorMsg
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unknown",
		})
	}
}
