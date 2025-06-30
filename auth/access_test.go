package auth

import (
	"druid-insight/config"
	"testing"
)

func makeTestUsersFile() *UsersFile {
	return &UsersFile{
		Users: map[string]UserInfo{
			"alice": {
				Admin: false,
				Access: map[string]map[string][]string{
					"myreport": {
						"browser": {"Chrome", "Firefox"},
						"country": {"FR"},
					},
				},
			},
			"bob": {
				Admin: true,
			},
		},
	}
}

func makeTestDruidConfig() *config.DruidConfig {
	return &config.DruidConfig{
		Datasources: map[string]config.DruidDatasourceSchema{
			"myreport": {
				Dimensions: map[string]config.DruidField{
					"browser": {Druid: "browser"},
					"country": {Druid: "country"},
					"device":  {Druid: "device"},
				},
				Metrics: map[string]config.DruidField{
					"requests": {Druid: "requests"},
				},
			},
		},
	}
}

func TestGetAccessFilters_AdminReturnsNil(t *testing.T) {
	cfg := &Config{}
	users := makeTestUsersFile()
	druidCfg := makeTestDruidConfig()
	filters := GetAccessFilters("bob", true, "myreport", druidCfg, users, cfg)
	if filters != nil {
		t.Errorf("Expected nil for admin user, got %v", filters)
	}
}

func TestGetAccessFilters_FileBackend(t *testing.T) {
	cfg := &Config{}
	users := makeTestUsersFile()
	druidCfg := makeTestDruidConfig()
	filters := GetAccessFilters("alice", false, "myreport", druidCfg, users, cfg)
	if len(filters) != 2 {
		t.Errorf("Expected 2 filters for alice, got %v", filters)
	}
	if v, ok := filters["browser"]; !ok || len(v) != 2 {
		t.Errorf("Expected browser filter with 2 values, got %v", v)
	}
	if v, ok := filters["country"]; !ok || len(v) != 1 || v[0] != "FR" {
		t.Errorf("Expected country filter with value FR, got %v", v)
	}
}

func TestGetAccessFilters_NoAccessInUsersFile(t *testing.T) {
	cfg := &Config{}
	users := makeTestUsersFile()
	druidCfg := makeTestDruidConfig()
	filters := GetAccessFilters("unknown", false, "myreport", druidCfg, users, cfg)
	if len(filters) != 0 {
		t.Errorf("Expected no filters for unknown user, got %v", filters)
	}
}

func TestGetAccessFilters_UnknownDatasource(t *testing.T) {
	cfg := &Config{}
	users := makeTestUsersFile()
	druidCfg := makeTestDruidConfig()
	filters := GetAccessFilters("alice", false, "unknown_ds", druidCfg, users, cfg)
	if len(filters) != 0 {
		t.Errorf("Expected no filters for unknown datasource, got %v", filters)
	}
}
