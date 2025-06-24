package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"druid-insight/config"
	"druid-insight/utils"
)

// Struct pour réponse SQL
type druidSQLCol struct {
	ColumnName string `json:"COLUMN_NAME"`
	DataType   string `json:"DATA_TYPE"`
}

func backupFile(yamlPath string) error {
	root := utils.GetProjectRoot()
	src := filepath.Join(root, yamlPath)
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}
	date := stat.ModTime().Format("20060102-1504")
	bakdir := filepath.Join(root, "archives")
	os.MkdirAll(bakdir, 0755)
	dst := filepath.Join(bakdir, fmt.Sprintf("druid.yaml.%s", date))
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func main() {
	var datasource string
	var dryRun bool
	var yamlFile string

	flag.StringVar(&datasource, "datasource", "", "Name of datasync to sync (required)")
	flag.BoolVar(&dryRun, "dry-run", false, "Simulate without update file")
	flag.StringVar(&yamlFile, "yaml", "druid.yaml", "Absolute yaml file path")
	flag.Parse()

	if datasource == "" {
		fmt.Println("Usage : datasource-sync --datasource <nom>")
		os.Exit(1)
	}

	// 1. Charger la config existante
	cfg, err := config.LoadDruidConfig(yamlFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed loading druid.yaml : %v\n", err)
		os.Exit(2)
	}

	// 2. Appel SQL Druid pour introspection
	endpoint := strings.TrimRight(cfg.HostURL, "/") + "/druid/v2/sql"
	sqlReq := map[string]interface{}{
		"query": fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = '%s'", datasource),
	}
	reqBody, _ := json.Marshal(sqlReq)
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed calling Druid SQL API : %v\n", err)
		os.Exit(2)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Failed Druid SQL : %s\n%s\n", resp.Status, body)
		os.Exit(2)
	}
	var columns []druidSQLCol
	if err := json.NewDecoder(resp.Body).Decode(&columns); err != nil {
		fmt.Fprintf(os.Stderr, "Failed parsing JSON SQL : %v\n", err)
		os.Exit(2)
	}

	// 3. Classement dimensions / metrics
	isText := func(t string) bool {
		t = strings.ToUpper(t)
		return strings.Contains(t, "VARCHAR") || strings.Contains(t, "STRING") || strings.Contains(t, "CHAR")
	}
	isNumber := func(t string) bool {
		t = strings.ToUpper(t)
		return strings.Contains(t, "BIGINT") || strings.Contains(t, "DOUBLE") ||
			strings.Contains(t, "FLOAT") || strings.Contains(t, "DECIMAL") ||
			strings.Contains(t, "LONG") || strings.Contains(t, "INTEGER") || strings.Contains(t, "INT")
	}

	dims := []string{}
	mets := []string{}
	for _, col := range columns {
		if isText(col.DataType) {
			dims = append(dims, col.ColumnName)
		} else if isNumber(col.DataType) {
			mets = append(mets, col.ColumnName)
		}
	}

	// 4. Mettre à jour la structure
	if cfg.Datasources == nil {
		cfg.Datasources = make(map[string]config.DruidDatasourceSchema, 0)
	}
	ds, exists := cfg.Datasources[datasource]
	if !exists {
		ds = config.DruidDatasourceSchema{
			Dimensions: map[string]config.DruidField{},
			Metrics:    map[string]config.DruidField{},
		}
	}

	var newDims, newMetrics []string

	for _, dim := range dims {
		if _, ok := ds.Dimensions[dim]; !ok {
			ds.Dimensions[dim] = config.DruidField{
				Druid:    dim,
				Reserved: false,
			}
			newDims = append(newDims, dim)
		}
	}
	for _, met := range mets {
		if _, ok := ds.Metrics[met]; !ok {
			ds.Metrics[met] = config.DruidField{
				Druid:    met,
				Type:     "line",
				Reserved: false,
			}
			newMetrics = append(newMetrics, met)
		}
	}

	cfg.Datasources[datasource] = ds

	if len(newDims) == 0 && len(newMetrics) == 0 {
		fmt.Println("No modification needed. Everything is up-to-date.")
	} else {
		fmt.Println("New entries summary :")
		if len(newDims) > 0 {
			fmt.Println("  Dimensions added :", strings.Join(newDims, ", "))
		}
		if len(newMetrics) > 0 {
			fmt.Println("  Metrics added    :", strings.Join(newMetrics, ", "))
		}
	}

	if !dryRun && (len(newDims) > 0 || len(newMetrics) > 0) {
		if err := backupFile(yamlFile); err != nil {
			fmt.Fprintf(os.Stderr, "Backup error : %v\n", err)
			os.Exit(2)
		}
		root := utils.GetProjectRoot()
		dst := filepath.Join(root, yamlFile)
		yamlOut, err := yaml.Marshal(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Marshal YAML error : %v\n", err)
			os.Exit(2)
		}
		if err := os.WriteFile(dst, yamlOut, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Writing YAML error : %v\n", err)
			os.Exit(2)
		}
		fmt.Println("Update done. Backup send to archives/")
	} else if dryRun && (len(newDims) > 0 || len(newMetrics) > 0) {
		fmt.Println("\n--- YAML would be : ---\n")
		out, _ := yaml.Marshal(cfg)
		fmt.Println(string(out))
	}
}
