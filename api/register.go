package api

import (
	"druid-insight/auth"
	"druid-insight/druid"
	"druid-insight/logging"
	"druid-insight/utils"
	"net/http"
)

func RegisterHandlers(cfg *auth.Config, users *auth.UsersFile, druidCfg *druid.DruidConfig, accessLogger, loginLogger, reportLogger *logging.Logger) {
	utils.LogToFile("api.log")
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
