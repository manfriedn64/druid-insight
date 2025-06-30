package druid

import (
	"druid-insight/auth"
	"druid-insight/config"
	"reflect"
	"testing"
)

func makeTestDruidSchema() config.DruidDatasourceSchema {
	return config.DruidDatasourceSchema{
		Dimensions: map[string]config.DruidField{
			"browser": {Druid: "browser"},
			"device":  {Druid: "device"},
			"date":    {Druid: "__time"},
		},
		Metrics: map[string]config.DruidField{
			"requests": {Druid: "requests"},
			"errors":   {Druid: "errors"},
			"cpm":      {Formula: "1000 * revenue / impressions"},
		},
	}
}

func TestBuildAggsAndPostAggs_Simple(t *testing.T) {
	ds := makeTestDruidSchema()
	aggs, postAggs, err := BuildAggsAndPostAggs([]string{"requests"}, ds)
	if err != nil {
		t.Fatalf("BuildAggsAndPostAggs failed: %v", err)
	}
	if len(aggs) != 1 || aggs[0]["name"] != "requests" {
		t.Errorf("Expected one agg for 'requests', got %v", aggs)
	}
	if len(postAggs) != 0 {
		t.Errorf("Expected no postAggs, got %v", postAggs)
	}
}

func TestBuildAggsAndPostAggs_Formula(t *testing.T) {
	ds := makeTestDruidSchema()
	// Add required metrics for formula
	ds.Metrics["revenue"] = config.DruidField{Druid: "revenue"}
	ds.Metrics["impressions"] = config.DruidField{Druid: "impressions"}
	ds.Metrics["cpm"] = config.DruidField{Formula: "1000 * revenue / impressions"}

	aggs, postAggs, err := BuildAggsAndPostAggs([]string{"cpm"}, ds)
	if err != nil {
		t.Fatalf("BuildAggsAndPostAggs failed: %v", err)
	}
	if len(aggs) != 2 {
		t.Errorf("Expected 2 aggs for formula (revenue, impressions), got %v", aggs)
	}
	if len(postAggs) != 1 {
		t.Errorf("Expected 1 postAgg for cpm, got %v", postAggs)
	}
}

func TestMergeWithAccessFilters(t *testing.T) {
	ds := makeTestDruidSchema()
	userFilters := []interface{}{
		map[string]interface{}{"dimension": "browser", "values": []interface{}{"Chrome"}},
	}
	access := map[string][]string{
		"device": {"Mobile"},
	}
	result := MergeWithAccessFilters(userFilters, access, ds)
	if len(result) != 2 {
		t.Errorf("Expected 2 filters merged, got %v", result)
	}
}

func TestConvertFiltersToDruidDimFilter(t *testing.T) {
	ds := makeTestDruidSchema()
	filters := []interface{}{
		map[string]interface{}{"dimension": "browser", "values": []interface{}{"Chrome", "Firefox"}},
		map[string]interface{}{"dimension": "device", "values": []interface{}{"Mobile"}},
	}
	filter := ConvertFiltersToDruidDimFilter(filters, ds)
	m, ok := filter.(map[string]interface{})
	if !ok || m["type"] != "and" {
		t.Errorf("Expected 'and' filter, got %v", filter)
	}
	fields, ok := m["fields"].([]interface{})
	if !ok || len(fields) != 2 {
		t.Errorf("Expected 2 fields in 'and', got %v", fields)
	}
}

func TestConvertFiltersToDruidDimFilter_Single(t *testing.T) {
	ds := makeTestDruidSchema()
	filters := []interface{}{
		map[string]interface{}{"dimension": "browser", "values": []interface{}{"Chrome"}},
	}
	filter := ConvertFiltersToDruidDimFilter(filters, ds)
	m, ok := filter.(map[string]interface{})
	if !ok || m["type"] != "in" {
		t.Errorf("Expected single 'in' filter, got %v", filter)
	}
	if m["dimension"] != "browser" {
		t.Errorf("Expected dimension 'browser', got %v", m["dimension"])
	}
}

func TestConvertFiltersToDruidDimFilter_Empty(t *testing.T) {
	ds := makeTestDruidSchema()
	filters := []interface{}{}
	filter := ConvertFiltersToDruidDimFilter(filters, ds)
	if filter != nil {
		t.Errorf("Expected nil for empty filters, got %v", filter)
	}
}

func TestBuildDruidQuery_UnknownDimension(t *testing.T) {
	ds := makeTestDruidSchema()
	cfg := &auth.Config{}
	druidCfg := &config.DruidConfig{
		Datasources: map[string]config.DruidDatasourceSchema{"myds": ds},
	}
	_, err := BuildDruidQuery("myds", []string{"unknown"}, []string{"requests"}, nil, nil, ds, "all", "alice", false, druidCfg, cfg, "")
	if err == nil {
		t.Error("Expected error for unknown dimension, got nil")
	}
}

func TestBuildDruidQuery_Basic(t *testing.T) {
	ds := makeTestDruidSchema()
	cfg := &auth.Config{}
	druidCfg := &config.DruidConfig{
		Datasources: map[string]config.DruidDatasourceSchema{"myds": ds},
	}
	query, err := BuildDruidQuery("myds", []string{"browser"}, []string{"requests"}, nil, nil, ds, "all", "alice", false, druidCfg, cfg, "")
	if err != nil {
		t.Fatalf("BuildDruidQuery failed: %v", err)
	}
	if query["queryType"] != "groupBy" {
		t.Errorf("Expected queryType 'groupBy', got %v", query["queryType"])
	}
	if !reflect.DeepEqual(query["dimensions"], []interface{}{"browser"}) {
		t.Errorf("Expected dimensions ['browser'], got %v", query["dimensions"])
	}
}
