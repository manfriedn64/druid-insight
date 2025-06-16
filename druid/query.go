package druid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// BuildAggsAndPostAggs analyse la liste des metrics demandées,
// et construit la liste des aggs et postAggs nécessaires.
func BuildAggsAndPostAggs(metrics []string, ds DruidDatasourceSchema) (aggs []map[string]interface{}, postAggs []map[string]interface{}, err error) {
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
				if !aggsSet[f] {
					aggs = append(aggs, map[string]interface{}{
						"type":      "doubleSum", // TODO: rendre dynamique selon métrique
						"name":      f,
						"fieldName": base.Druid,
					})
					aggsSet[f] = true
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
func BuildDruidQuery(dsName string, dims []string, mets []string, filters interface{}, intervals []string, ds DruidDatasourceSchema) (map[string]interface{}, error) {
	var druidDims []string
	for _, d := range dims {
		dr, ok := ds.Dimensions[d]
		if !ok {
			return nil, fmt.Errorf("unknown dimension: %s", d)
		}
		druidDims = append(druidDims, dr.Druid)
	}
	aggs, postAggs, err := BuildAggsAndPostAggs(mets, ds)
	if err != nil {
		return nil, err
	}
	query := map[string]interface{}{
		"queryType":    "groupBy",
		"dataSource":   dsName,
		"dimensions":   druidDims,
		"granularity":  "all",
		"aggregations": aggs,
	}
	if len(postAggs) > 0 {
		query["postAggregations"] = postAggs
	}
	if filters != nil {
		query["filter"] = filters
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
func ConvertFiltersToDruidDimFilter(filters []interface{}, ds DruidDatasourceSchema) interface{} {
	var filterFields []interface{}
	for _, f := range filters {
		fmap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		dimKey, _ := fmap["dimension"].(string)
		values, _ := fmap["values"].([]interface{})
		// Convertit les valeurs en []string
		var svalues []string
		for _, v := range values {
			if sv, ok := v.(string); ok {
				svalues = append(svalues, sv)
			}
		}
		// Récupère le vrai nom Druid
		dimName := ds.Dimensions[dimKey].Druid
		filterFields = append(filterFields, map[string]interface{}{
			"type":      "in",
			"dimension": dimName,
			"values":    svalues,
		})
	}
	if len(filterFields) == 0 {
		return nil
	}
	if len(filterFields) == 1 {
		return filterFields[0]
	}
	// Si plusieurs, les combiner en AND
	return map[string]interface{}{
		"type":   "and",
		"fields": filterFields,
	}
}
