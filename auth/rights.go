package auth

import "druid-insight/config"

func CheckRights(payload map[string]interface{}, druidCfg *config.DruidConfig, datasource string, isAdmin bool) []string {
	problems := []string{}
	ds, ok := druidCfg.Datasources[datasource]
	if !ok {
		return []string{"datasource_not_found"}
	}
	if dims, ok := payload["dimensions"].([]interface{}); ok {
		for _, dimRaw := range dims {
			dim, _ := dimRaw.(string)
			if dim == "time" {
				// La dimension "time" est TOUJOURS autorisée
				continue
			}
			f, ok := ds.Dimensions[dim]
			if !ok {
				problems = append(problems, "dimension:"+dim+":unknown")
			} else if f.Reserved && !isAdmin {
				problems = append(problems, "dimension:"+dim+":forbidden")
			}
		}
	}
	if mets, ok := payload["metrics"].([]interface{}); ok {
		for _, mRaw := range mets {
			metric, _ := mRaw.(string)
			f, ok := ds.Metrics[metric]
			if !ok {
				problems = append(problems, "metric:"+metric+":unknown")
			} else if f.Reserved && !isAdmin {
				problems = append(problems, "metric:"+metric+":forbidden")
			}
		}
	}
	return problems
}
