package auth

import (
	"database/sql"
	"druid-insight/config"
	"log"
)

func GetAccessFilters(username string, isAdmin bool, datasource string, druidCfg *config.DruidConfig, users *UsersFile, cfg *Config) map[string][]string {
	if isAdmin {
		return nil
	}

	if users != nil {
		return getFiltersFromFile(username, datasource, druidCfg, users)
	} else {
		return getFiltersFromDB(username, druidCfg, cfg)
	}
}

func getFiltersFromFile(username string, datasource string, druidCfg *config.DruidConfig, users *UsersFile) map[string][]string {
	result := make(map[string][]string, 0)

	ds, ok := druidCfg.Datasources[datasource]
	if !ok {
		return result
	}

	for dimName := range ds.Dimensions {
		var values []string

		// 1. Vérifie dans users.yaml
		if users != nil {
			if userInfo, ok := users.Users[username]; ok {
				if accessByDS, ok := userInfo.Access[datasource]; ok {
					if vals, ok := accessByDS[dimName]; ok && len(vals) > 0 {
						values = append(values, vals...)
					}
				}
			}
		}

		if len(values) > 0 {
			result[dimName] = values
		}
	}
	return result
}

func getFiltersFromDB(user string, druidCfg *config.DruidConfig, cfg *Config) map[string][]string {
	var (
		db      *sql.DB
		err     error
		value   string
		filters = make(map[string][]string, 0)
	)

	if cfg.Auth.UserBackend == "file" {
		return nil
	}

	db, err = sql.Open(cfg.Auth.UserBackend, cfg.Auth.DBDSN)
	if err != nil {
		log.Fatal("could not load db for UserSetFilters - " + err.Error())
	} else {
		defer db.Close()
	}

	for datasource := range druidCfg.Datasources {
		for dims := range druidCfg.Datasources[datasource].Dimensions {
			if len(druidCfg.Datasources[datasource].Dimensions[dims].AccessQuery) > 0 {
				if rows, err := db.Query(druidCfg.Datasources[datasource].Dimensions[dims].AccessQuery, user); err == nil {
					filters[dims] = make([]string, 0)
					for rows.Next() {
						err = rows.Scan(&value)
						if err == nil {
							filters[dims] = append(filters[dims], value)
						}
					}
				} else {
					log.Println("UserSetFilters - " + err.Error())
				}
			}
		}
	}
	return filters
}
