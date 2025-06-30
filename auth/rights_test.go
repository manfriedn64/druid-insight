package auth

import (
	"druid-insight/config"
	"testing"
)

func makeDruidConfig() *config.DruidConfig {
	return &config.DruidConfig{
		HostURL: "http://localhost:8082/query",
		Datasources: map[string]config.DruidDatasourceSchema{
			"myreport": {
				Dimensions: map[string]config.DruidField{
					"date":    {Druid: "__time", Reserved: false},
					"device":  {Druid: "device", Reserved: false},
					"browser": {Druid: "browser", Reserved: true},
				},
				Metrics: map[string]config.DruidField{
					"requests": {Druid: "requests", Reserved: false},
					"errors":   {Druid: "errors", Reserved: true},
				},
			},
		},
	}
}

func TestCheckRights_AllOk_Admin(t *testing.T) {
	payload := map[string]interface{}{
		"dimensions": []interface{}{"date", "browser"},
		"metrics":    []interface{}{"requests", "errors"},
	}
	problems := CheckRights(payload, makeDruidConfig(), "myreport", true)
	if len(problems) != 0 {
		t.Errorf("Expected no problems for admin, got: %v", problems)
	}
}

func TestCheckRights_ForbiddenForUser(t *testing.T) {
	payload := map[string]interface{}{
		"dimensions": []interface{}{"date", "browser"},
		"metrics":    []interface{}{"requests", "errors"},
	}
	problems := CheckRights(payload, makeDruidConfig(), "myreport", false)
	expected := map[string]bool{
		"dimension:browser:forbidden": true,
		"metric:errors:forbidden":     true,
	}
	if len(problems) != 2 {
		t.Errorf("Expected 2 problems, got %v", problems)
	}
	for _, p := range problems {
		if !expected[p] {
			t.Errorf("Unexpected problem: %v", p)
		}
	}
}

func TestCheckRights_UnknownDimensionAndMetric(t *testing.T) {
	payload := map[string]interface{}{
		"dimensions": []interface{}{"date", "unknown_dim"},
		"metrics":    []interface{}{"requests", "unknown_metric"},
	}
	problems := CheckRights(payload, makeDruidConfig(), "myreport", false)
	expected := map[string]bool{
		"dimension:unknown_dim:unknown": true,
		"metric:unknown_metric:unknown": true,
	}
	if len(problems) != 2 {
		t.Errorf("Expected 2 problems, got %v", problems)
	}
	for _, p := range problems {
		if !expected[p] {
			t.Errorf("Unexpected problem: %v", p)
		}
	}
}

func TestCheckRights_DatasourceNotFound(t *testing.T) {
	payload := map[string]interface{}{
		"dimensions": []interface{}{"date"},
		"metrics":    []interface{}{"requests"},
	}
	problems := CheckRights(payload, makeDruidConfig(), "unknown_ds", false)
	if len(problems) != 1 || problems[0] != "datasource_not_found" {
		t.Errorf("Expected datasource_not_found, got %v", problems)
	}
}

func TestCheckRights_TimeAlwaysAllowed(t *testing.T) {
	payload := map[string]interface{}{
		"dimensions": []interface{}{"time"},
		"metrics":    []interface{}{"requests"},
	}
	problems := CheckRights(payload, makeDruidConfig(), "myreport", false)
	if len(problems) != 0 {
		t.Errorf("Expected no problems for dimension 'time', got %v", problems)
	}
}
