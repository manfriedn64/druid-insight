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
	"druid-insight/config"
)

type filterCache struct {
	Values    []string
	ExpiresAt time.Time
}

var filterMemoryCache sync.Map // key = datasource|dimension, value = filterCache

type FilterRequest struct {
	Datasource string `json:"datasource"`
	Dimension  string `json:"dimension"`
	DateStart  string `json:"date_start,omitempty"`
	DateEnd    string `json:"date_end,omitempty"`
}

type FilterResponse struct {
	Values []string `json:"values"`
}

func GetDimensionValues(cfg *auth.Config, druidCfg *config.DruidConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, isAdmin, err := auth.ExtractUserAndAdminFromJWT(r, cfg.JWT.Secret)
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

		datePart := ""
		if filterReq.DateStart != "" && filterReq.DateEnd != "" {
			datePart = "|" + filterReq.DateStart + "|" + filterReq.DateEnd
		}
		cacheKey := username + "|" + filterReq.Datasource + "|" + druidDimension.Druid + datePart
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

		// Charger les droits locaux
		usersFile, _ := auth.LoadUsers(cfg.Auth.UserFile)
		accessFilters := auth.GetAccessFilters(username, isAdmin, filterReq.Datasource, druidCfg, usersFile, cfg)

		var druidFilter interface{} = nil
		if accessFilters != nil {
			if len(accessFilters) == 1 {
				for dim := range accessFilters {
					druidFilter = map[string]interface{}{
						"type":      "in",
						"dimension": dim,
						"values":    accessFilters[dim],
					}
				}
			} else {
				fields := make([]map[string]interface{}, 0)
				for dim := range accessFilters {
					temp := map[string]interface{}{
						"type":      "in",
						"dimension": dim,
						"values":    accessFilters[dim],
					}
					fields = append(fields, temp)
				}
				druidFilter = map[string]interface{}{
					"type":   "and",
					"fields": fields,
				}
			}
		}

		intervals := []string{"1000-01-01T00:00:00.000Z/3000-01-01T00:00:00.000Z"}
		if filterReq.DateStart != "" && filterReq.DateEnd != "" {
			intervals = []string{filterReq.DateStart + "T00:00:00.000Z/" + filterReq.DateEnd + "T23:59:59.999Z"}
		}

		druidQuery := map[string]interface{}{
			"context":     map[string]string{"application": "druid-insight"},
			"queryType":   "groupBy",
			"dataSource":  dsConfig.DruidName,
			"dimensions":  []string{druidDimension.Druid},
			"granularity": "all",
			"intervals":   intervals,
		}

		if druidFilter != nil {
			druidQuery["filter"] = druidFilter
		}

		queryBytes, err := json.Marshal(druidQuery)
		if err != nil {
			http.Error(w, "Failed to create Druid query", http.StatusInternalServerError)
			return
		}
		log.Println(string(queryBytes))
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
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Failed to fetch data from Druid", http.StatusInternalServerError)
			log.Println(string(body))
			return
		}

		if err != nil {
			http.Error(w, "Failed to read response from Druid", http.StatusInternalServerError)
			return
		}

		var druidResp []struct {
			Event map[string]interface{} `json:"event"`
		}

		if err := json.Unmarshal(body, &druidResp); err != nil {
			http.Error(w, "Invalid response from Druid", http.StatusInternalServerError)
			return
		}

		valuesSet := make(map[string]struct{})
		for _, entry := range druidResp {
			valRaw, ok := entry.Event[druidDimension.Druid]
			if !ok || valRaw == nil {
				continue
			}
			val, ok := valRaw.(string)
			if !ok {
				val = fmt.Sprintf("%v", valRaw)
			}
			valuesSet[val] = struct{}{}
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
