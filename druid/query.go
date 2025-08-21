package druid

import (
	"bytes"
	"database/sql"
	"druid-insight/auth"
	"druid-insight/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
)

// BuildAggsAndPostAggs analyse la liste des metrics demandées,
// et construit la liste des aggs et postAggs nécessaires.
func BuildAggsAndPostAggs(metrics []string, ds config.DruidDatasourceSchema) (aggs []map[string]interface{}, postAggs []map[string]interface{}, err error) {
	aggsSet := map[string]bool{}
	for _, m := range metrics {
		mf := ds.Metrics[m]
		if mf.Formula != "" {
			node, err2 := ParseFormula(mf.Formula)
			if err2 != nil {
				return nil, nil, fmt.Errorf("parse formula for %s: %w", m, err2)
			}
			leaves := CollectLeafFields(node)
			for _, f := range leaves {
				base, ok := ds.Metrics[f]
				if !ok || base.Druid == "" {
					return nil, nil, fmt.Errorf("metric %s used in formula %s not found", f, m)
				}
				aggName := "sum_" + f // convention pour sum(x)
				if !aggsSet[aggName] {
					aggs = append(aggs, map[string]interface{}{
						"type":      "doubleSum", // TODO: rendre dynamique selon métrique
						"name":      aggName,
						"fieldName": base.Druid,
					})
					aggsSet[aggName] = true
				}
			}
			postAggs = append(postAggs, NodeToDruidPostAgg(m, node))
		} else if mf.Druid != "" {
			if !aggsSet[m] {
				aggs = append(aggs, map[string]interface{}{
					"type":      "doubleSum",
					"name":      m,
					"fieldName": mf.Druid,
				})
				aggsSet[m] = true
			}
		}
	}
	return
}

// BuildDruidQuery construit la requête groupBy pour Druid (JSON map) à partir des inputs
func BuildDruidQuery(dsName string, dims []string, mets []string, userFilters interface{}, intervals []string, ds config.DruidDatasourceSchema, granularity string, username string, isAdmin bool, druidCfg *config.DruidConfig, cfg *auth.Config, owner string, context string) (map[string]interface{}, error) {
	var usersFile *auth.UsersFile

	var druidDims []interface{}
	for _, d := range dims {
		if d == "time" {
			switch granularity {
			case "month":
				druidDims = append(druidDims, map[string]interface{}{
					"type":       "extraction",
					"dimension":  "__time",
					"outputName": "time",
					"extractionFn": map[string]interface{}{
						"type":     "timeFormat",
						"format":   "yyyy-MM",
						"timeZone": "Europe/Paris",
					},
				})
			case "day":
				druidDims = append(druidDims, map[string]interface{}{
					"type":       "extraction",
					"dimension":  "__time",
					"outputName": "time",
					"extractionFn": map[string]interface{}{
						"type":     "timeFormat",
						"format":   "yyyy-MM-dd",
						"timeZone": "Europe/Paris",
					},
				})
			case "hour":
				druidDims = append(druidDims, map[string]interface{}{
					"type":       "extraction",
					"dimension":  "__time",
					"outputName": "time",
					"extractionFn": map[string]interface{}{
						"type":     "timeFormat",
						"format":   "yyyy-MM-dd HH",
						"timeZone": "Europe/Paris",
					},
				})
			case "week":
				druidDims = append(druidDims, map[string]interface{}{
					"type":       "extraction",
					"dimension":  "__time",
					"outputName": "time",
					"extractionFn": map[string]interface{}{
						"type":     "timeFormat",
						"format":   "YYYY-'W'ww", // ISO semaine, à adapter selon besoin
						"timeZone": "Europe/Paris",
					},
				})
			default:
				druidDims = append(druidDims, map[string]interface{}{
					"type":       "default",
					"dimension":  "__time",
					"outputName": "time",
				})
			}
			continue
		}
		dr, ok := ds.Dimensions[d]
		if !ok {
			return nil, fmt.Errorf("unknown dimension: %s", d)
		}
		if dr.Lookup != "" {
			druidDims = append(druidDims, map[string]interface{}{
				"type":       "lookup",
				"dimension":  dr.Druid,
				"outputName": d,
				"name":       dr.Lookup,
			})
		} else {
			druidDims = append(druidDims, dr.Druid)
		}
	}
	aggs, postAggs, err := BuildAggsAndPostAggs(mets, ds)
	if err != nil {
		return nil, err
	}
	g := granularity
	if g == "" {
		g = "all"
	}

	// 2. Construire la requête groupBy via BuildDruidQuery
	if cfg.Auth.UserBackend == "file" {
		usersFile, _ = auth.LoadUsers("config/users.yaml")
	} else if slices.Contains([]string{"mysql", "postgres", "sqlite"}, cfg.Auth.UserBackend) {
		db, err := sql.Open(cfg.Auth.UserBackend, cfg.Auth.DBDSN)
		if err == nil {
			defer db.Close()
		}
	}

	// Appliquer les restrictions d'accès utilisateur
	accessFilters := auth.GetAccessFilters(username, isAdmin, dsName, druidCfg, usersFile, cfg)
	combinedFilters := MergeWithAccessFilters(userFilters, accessFilters, ds)
	druidDimFilter := ConvertFiltersToDruidDimFilter(combinedFilters, ds)

	query := map[string]interface{}{
		"context":      map[string]string{"application": context},
		"queryType":    "groupBy",
		"dataSource":   ds.DruidName,
		"dimensions":   druidDims,
		"granularity":  g,
		"aggregations": aggs,
	}
	if len(postAggs) > 0 {
		query["postAggregations"] = postAggs
	}
	if druidDimFilter != nil {
		query["filter"] = druidDimFilter
	}
	if len(intervals) > 0 {
		query["intervals"] = intervals
	}
	return query, nil
}

// ExecuteDruidQuery exécute la requête groupBy sur Druid, et retourne le résultat.
func ExecuteDruidQuery(hostURL string, query map[string]interface{}) ([]map[string]interface{}, error) {
	j, _ := json.Marshal(query)
	log.Println("execute query : " + string(j))
	resp, err := http.Post(hostURL, "application/json", bytes.NewReader(j))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		bb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("druid HTTP %d: %s", resp.StatusCode, string(bb))
	}
	var res []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res, nil
}

// filters doit être []interface{}, chaque élément étant map[string]interface{} avec "dimension" et "values"
// ds: DruidDatasourceSchema pour récupérer le vrai nom Druid
func ConvertFiltersToDruidDimFilter(filters []interface{}, ds config.DruidDatasourceSchema) interface{} {
	var filterFields []interface{}
	for _, f := range filters {
		fmap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		dimKey, _ := fmap["dimension"].(string)
		values, _ := fmap["values"].([]interface{})
		var svalues []string
		for _, v := range values {
			if sv, ok := v.(string); ok {
				svalues = append(svalues, sv)
			}
		}
		field := ds.Dimensions[dimKey]
		if field.Lookup != "" {
			filterFields = append(filterFields, map[string]interface{}{
				"type":      "in",
				"dimension": field.Druid,
				"values":    svalues,
				"extractionFn": map[string]interface{}{
					"type":   "lookup",
					"lookup": field.Lookup,
				},
			})
		} else {
			filterFields = append(filterFields, map[string]interface{}{
				"type":      "in",
				"dimension": field.Druid,
				"values":    svalues,
			})
		}
	}
	if len(filterFields) == 0 {
		return nil
	}
	if len(filterFields) == 1 {
		return filterFields[0]
	}
	return map[string]interface{}{
		"type":   "and",
		"fields": filterFields,
	}
}

func MergeWithAccessFilters(userFilters interface{}, access map[string][]string, ds config.DruidDatasourceSchema) []interface{} {
	result := []interface{}{}

	if userFilters != nil {
		if arr, ok := userFilters.([]interface{}); ok {
			result = append(result, arr...)
		}
	}

	for dim, vals := range access {
		if len(vals) == 0 {
			continue
		}
		values := []interface{}{}
		for _, v := range vals {
			values = append(values, v)
		}
		result = append(result, map[string]interface{}{
			"dimension": dim,
			"values":    values,
		})
	}
	return result
}
