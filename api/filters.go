package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

	"druid-insight/auth"
	"druid-insight/druid"
)

type filterCache struct {
	Values    []string
	ExpiresAt time.Time
}

var filterMemoryCache sync.Map // key = datasource|dimension, value = filterCache

type FilterRequest struct {
	Datasource string `json:"datasource"`
	Dimension  string `json:"dimension"`
}

type FilterResponse struct {
	Values []string `json:"values"`
}

func GetDimensionValues(cfg *auth.Config, druidCfg *druid.DruidConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, isAdmin, err := auth.ExtractUserAndAdminFromJWT(r, cfg.JWT.Secret)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		var filterReq FilterRequest
		if err := json.NewDecoder(r.Body).Decode(&filterReq); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if filterReq.Dimension == "" || filterReq.Datasource == "" {
			http.Error(w, "Dimension and Datasource are required", http.StatusBadRequest)
			return
		}

		dsConfig, ok := druidCfg.Datasources[filterReq.Datasource]
		if !ok {
			http.Error(w, "Datasource not found in configuration", http.StatusBadRequest)
			return
		}

		druidDimension, ok := dsConfig.Dimensions[filterReq.Dimension]
		if !ok {
			http.Error(w, "Dimension not found in configuration", http.StatusBadRequest)
			return
		}

		if !isAdmin && druidDimension.Reserved {
			http.Error(w, "Forbidden: access denied to dimension", http.StatusForbidden)
			return
		}

		cacheKey := filterReq.Datasource + "|" + druidDimension.Druid
		now := time.Now()
		if val, found := filterMemoryCache.Load(cacheKey); found {
			cache := val.(filterCache)
			if cache.ExpiresAt.After(now) {
				log.Printf("filters.go - load %s from cache \n", cacheKey)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(FilterResponse{Values: cache.Values})
				return
			} else {
				log.Printf("filters.go - expired cache for %s \n", cacheKey)
			}
		}

		druidQuery := map[string]interface{}{
			"queryType":    "scan",
			"dataSource":   filterReq.Datasource,
			"columns":      []string{druidDimension.Druid},
			"resultFormat": "compactedList",
			"limit":        1000000,
			"intervals":    []string{"1000-01-01T00:00:00.000Z/3000-01-01T00:00:00.000Z"},
		}

		queryBytes, err := json.Marshal(druidQuery)
		if err != nil {
			http.Error(w, "Failed to create Druid query", http.StatusInternalServerError)
			return
		}
		log.Printf("filters.go - %s not in cache, calling api \n", cacheKey)
		req, err := http.NewRequest("POST", druidCfg.HostURL+"/druid/v2/", bytes.NewBuffer(queryBytes))
		if err != nil {
			http.Error(w, "Failed to create request to Druid", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Failed to fetch data from Druid", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read response from Druid", http.StatusInternalServerError)
			return
		}

		var druidResp []struct {
			Columns []string        `json:"columns"`
			Events  [][]interface{} `json:"events"`
		}
		if err := json.Unmarshal(body, &druidResp); err != nil {
			http.Error(w, "Invalid response from Druid", http.StatusInternalServerError)
			return
		}

		colIdx := -1
		for i, col := range druidResp[0].Columns {
			if col == druidDimension.Druid {
				colIdx = i
				break
			}
		}
		if colIdx == -1 {
			http.Error(w, "Column not found in Druid response", http.StatusInternalServerError)
			return
		}
		valuesSet := make(map[string]struct{})
		for _, entry := range druidResp {
			for _, evt := range entry.Events {
				if colIdx < len(evt) {
					val, ok := evt[colIdx].(string)
					if !ok {
						val = fmt.Sprintf("%v", evt[colIdx])
					}
					valuesSet[val] = struct{}{}
				}
			}
		}
		var values []string
		for val := range valuesSet {
			values = append(values, val)
		}
		slices.Sort(values)

		filterMemoryCache.Store(cacheKey, filterCache{
			Values:    values,
			ExpiresAt: now.Add(time.Hour),
		})
		log.Printf("filters.go - now in cache : %s \n", cacheKey)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FilterResponse{Values: values})
	}
}
