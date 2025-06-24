package api

import (
	"druid-insight/auth"
	"druid-insight/config"
	"druid-insight/logging"
	"encoding/json"
	"net/http"
	"sort"
)

func SchemaHandler(cfg *auth.Config, druidCfg *config.DruidConfig, accessLogger *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, isAdmin, err := auth.ExtractUserAndAdminFromJWT(r, cfg.JWT.Secret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		accessLogger.Write("SCHEMA user=" + user)

		type MetricObj struct {
			Name string `json:"name"`
			Type string `json:"type,omitempty"`
		}
		type dsObj struct {
			Dimensions []string    `json:"dimensions"`
			Metrics    []MetricObj `json:"metrics"`
		}
		schema := map[string]dsObj{}

		dsNames := make([]string, 0, len(druidCfg.Datasources))
		for name := range druidCfg.Datasources {
			dsNames = append(dsNames, name)
		}
		sort.Strings(dsNames)

		for _, dsName := range dsNames {
			ds := druidCfg.Datasources[dsName]
			var dims []string
			var mets []MetricObj

			for k, v := range ds.Dimensions {
				if !v.Reserved || isAdmin {
					dims = append(dims, k)
				}
			}
			sort.Strings(dims)

			var metNames []string
			metType := make(map[string]string)
			for k, v := range ds.Metrics {
				if !v.Reserved || isAdmin {
					metNames = append(metNames, k)
					metType[k] = v.Type
				}
			}
			sort.Strings(metNames)
			for _, mn := range metNames {
				mets = append(mets, MetricObj{Name: mn, Type: metType[mn]})
			}

			schema[dsName] = dsObj{Dimensions: dims, Metrics: mets}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(schema)
	}
}
