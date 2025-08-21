package config

import (
	"druid-insight/utils"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type DruidConfig struct {
	HostURL     string                           `yaml:"host_url"`
	Datasources map[string]DruidDatasourceSchema `yaml:"datasources"`
}

type DruidDatasourceSchema struct {
	DruidName  string                `yaml:"druid_name"` // nom réel dans Druid
	Dimensions map[string]DruidField `yaml:"dimensions"`
	Metrics    map[string]DruidField `yaml:"metrics"`
}

type DruidField struct {
	Druid       string `yaml:"druid"`
	Formula     string `yaml:"formula,omitempty"`
	Reserved    bool   `yaml:"reserved"`
	Type        string `yaml:"type,omitempty"`         // "bar" or "line"
	AccessQuery string `yaml:"access_query,omitempty"` // nouvelle ligne
	Lookup      string `yaml:"lookup,omitempty"`       // nom du lookup druid (optionnel)
}

func LoadDruidConfig(file string) (*DruidConfig, error) {
	var cfg DruidConfig
	root := utils.GetProjectRoot()
	cfgPath := filepath.Join(root, file)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
