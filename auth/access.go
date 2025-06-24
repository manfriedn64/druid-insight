package auth

import (
	"database/sql"
	"druid-insight/config"
)

func GetAccessFilters(username string, isAdmin bool, datasource string, druidCfg *config.DruidConfig, users *UsersFile, db *sql.DB) map[string][]string {
	if isAdmin {
		return nil
	}
	result := map[string][]string{}

	ds, ok := druidCfg.Datasources[datasource]
	if !ok {
		return result
	}

	for dimName, field := range ds.Dimensions {
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

		// 2. Sinon, vérifie via access_query
		if len(values) == 0 && db != nil && field.AccessQuery != "" {
			rows, err := db.Query(field.AccessQuery, username)
			if err != nil {
				continue
			}
			defer rows.Close()
			for rows.Next() {
				var val string
				if err := rows.Scan(&val); err == nil {
					values = append(values, val)
				}
			}
		}

		if len(values) > 0 {
			result[dimName] = values
		}
	}

	return result
}
