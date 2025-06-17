package main

import (
	"druid-insight/api"
	"druid-insight/auth"
	"druid-insight/druid"
	"druid-insight/logging"
	"druid-insight/static"
	"druid-insight/utils"
	"druid-insight/worker"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	cfg      *auth.Config
	users    *auth.UsersFile
	druidCfg *druid.DruidConfig
	loggers  []*logging.Logger
)

func main() {
	utils.LogToFile("api.log")
	loadEverything()

	worker.StartReportWorkers(5, druidCfg, loggers[2])

	api.RegisterHandlers(cfg, users, druidCfg, loggers[0], loggers[1], loggers[2])
	static.RegisterStaticHandler(cfg, loggers[0])

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)
	go func() {
		for range sigs {
			log.Println("Reloading configs...")
			loadEverything()
		}
	}()

	log.Printf("Serveur started listening onr %s ...", cfg.Server.Listen)
	log.Fatal(api.StartServer(cfg.Server.Listen))
}

func loadEverything() {
	var err error
	cfg, err = auth.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed config.yaml: %v", err)
	}
	users, err = auth.LoadUsers(cfg.Auth.UserFile)
	if err != nil {
		log.Fatalf("Failed users.yaml: %v", err)
	}
	druidCfg, err = druid.LoadDruidConfig("druid.yaml")
	if err != nil {
		log.Fatalf("Failed druid.yaml: %v", err)
	}
	os.MkdirAll(cfg.Server.LogDir, 0755)
	loggers = []*logging.Logger{
		logging.NewLoggerOrDie(cfg.Server.LogDir, "access.log"),
		logging.NewLoggerOrDie(cfg.Server.LogDir, "login.log"),
		logging.NewLoggerOrDie(cfg.Server.LogDir, "report.log"),
	}
}
