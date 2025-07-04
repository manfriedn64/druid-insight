package api

import (
	"druid-insight/auth"
	"druid-insight/config"
	"druid-insight/logging"
	"net/http"
)

func RegisterHandlers(cfg *auth.Config, users *auth.UsersFile, druidCfg *config.DruidConfig, accessLogger, loginLogger, reportLogger *logging.Logger) {
	http.HandleFunc("/api/login", LoginHandler(cfg, users, loginLogger))
	http.HandleFunc("/api/schema", SchemaHandler(cfg, druidCfg, accessLogger))
	http.HandleFunc("/api/reports/execute", ReportExecuteHandler(cfg, users, druidCfg, accessLogger))
	http.HandleFunc("/api/reports/status", ReportStatusHandler(cfg))
	http.HandleFunc("/api/reports/download", DownloadReportCSV(cfg))
	http.HandleFunc("/api/filters/values", GetDimensionValues(cfg, druidCfg))
}

func StartServer(listenAddr string) error {
	return http.ListenAndServe(listenAddr, nil)
}
