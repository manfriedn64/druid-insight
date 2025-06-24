package api

import (
	"druid-insight/auth"
	"druid-insight/config"
	"druid-insight/logging"
	"druid-insight/utils"
	"druid-insight/worker"
	"encoding/json"
	"net/http"
	"time"
)

func ReportExecuteHandler(cfg *auth.Config, users *auth.UsersFile, druidCfg *config.DruidConfig, accessLogger *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, isAdmin, err := auth.ExtractUserAndAdminFromJWT(r, cfg.JWT.Secret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if username == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			accessLogger.Write("EXECUTE_FAIL user=<unauth>")
			return
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			accessLogger.Write("EXECUTE_FAIL user=" + username + " bad_json")
			return
		}
		datasource, _ := payload["datasource"].(string)
		if datasource == "" {
			http.Error(w, "Datasource missing", http.StatusBadRequest)
			accessLogger.Write("EXECUTE_FAIL user=" + username + " missing_datasource")
			return
		}
		problems := auth.CheckRights(payload, druidCfg, datasource, isAdmin)
		if len(problems) > 0 {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":    "forbidden",
				"problems": problems,
			})
			accessLogger.Write("EXECUTE_FORBIDDEN user=" + username + " problems=" + jsonString(problems))
			return
		}
		id := utils.GenerateRequestID()
		req := &worker.ReportRequest{
			ID:         id,
			Payload:    payload,
			Owner:      username,
			Admin:      isAdmin,
			Datasource: datasource,
			CreatedAt:  time.Now(),
		}
		worker.AddPendingRequest(req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": id})
		accessLogger.Write("EXECUTE_OK user=" + username + " id=" + id)
	}
}

func jsonString(i interface{}) string {
	b, _ := json.Marshal(i)
	return string(b)
}
