package druid

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
	Dimensions map[string]DruidField `yaml:"dimensions"`
	Metrics    map[string]DruidField `yaml:"metrics"`
}

type DruidField struct {
	Druid    string `yaml:"druid"`
	Formula  string `yaml:"formula,omitempty"`
	Reserved bool   `yaml:"reserved"`
	Type     string `yaml:"type,omitempty"` // "bar" or "line"
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
