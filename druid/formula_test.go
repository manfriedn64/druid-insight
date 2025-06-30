package druid

import (
	"reflect"
	"testing"
)

func TestTokenizeFormula(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"1000 * revenue / impressions", []string{"1000", "*", "revenue", "/", "impressions"}},
		{"a+b-c", []string{"a", "+", "b", "-", "c"}},
		{"(a+b)*c", []string{"(", "a", "+", "b", ")", "*", "c"}},
		{"foo_1 + 2.5", []string{"foo_1", "+", "2.5"}},
	}
	for _, test := range tests {
		tokens, err := tokenizeFormula(test.input)
		if err != nil {
			t.Errorf("tokenizeFormula(%q) returned error: %v", test.input, err)
		}
		if !reflect.DeepEqual(tokens, test.expected) {
			t.Errorf("tokenizeFormula(%q) = %v, want %v", test.input, tokens, test.expected)
		}
	}
}

func TestTokenizeFormula_Invalid(t *testing.T) {
	_, err := tokenizeFormula("foo@bar")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}

func TestParseFormula_Simple(t *testing.T) {
	node, err := ParseFormula("1000 * revenue / impressions")
	if err != nil {
		t.Fatalf("ParseFormula failed: %v", err)
	}
	// Check root node
	if node.Op != "/" {
		t.Errorf("Expected root op '/', got %q", node.Op)
	}
	if node.Left.Op != "*" {
		t.Errorf("Expected left op '*', got %q", node.Left.Op)
	}
	if node.Left.Left.Value != "1000" || node.Left.Right.Value != "revenue" {
		t.Errorf("Unexpected left subtree: %+v", node.Left)
	}
	if node.Right.Value != "impressions" {
		t.Errorf("Unexpected right value: %v", node.Right.Value)
	}
}

func TestParseFormula_Parentheses(t *testing.T) {
	node, err := ParseFormula("(a+b)*c")
	if err != nil {
		t.Fatalf("ParseFormula failed: %v", err)
	}
	if node.Op != "*" {
		t.Errorf("Expected root op '*', got %q", node.Op)
	}
	if node.Left.Op != "+" {
		t.Errorf("Expected left op '+', got %q", node.Left.Op)
	}
	if node.Left.Left.Value != "a" || node.Left.Right.Value != "b" {
		t.Errorf("Unexpected left subtree: %+v", node.Left)
	}
	if node.Right.Value != "c" {
		t.Errorf("Unexpected right value: %v", node.Right.Value)
	}
}

func TestParseFormula_Invalid(t *testing.T) {
	_, err := ParseFormula("a + (b * c")
	if err == nil {
		t.Error("Expected error for missing parenthesis, got nil")
	}
	_, err = ParseFormula("a + )")
	if err == nil {
		t.Error("Expected error for unexpected parenthesis, got nil")
	}
}

func TestCollectLeafFields(t *testing.T) {
	node, err := ParseFormula("1000 * revenue / impressions")
	if err != nil {
		t.Fatalf("ParseFormula failed: %v", err)
	}
	leafs := CollectLeafFields(node)
	expected := []string{"revenue", "impressions"}
	if !reflect.DeepEqual(leafs, expected) {
		t.Errorf("CollectLeafFields = %v, want %v", leafs, expected)
	}
}

func TestNodeToDruidPostAgg(t *testing.T) {
	node, err := ParseFormula("1000 * revenue / impressions")
	if err != nil {
		t.Fatalf("ParseFormula failed: %v", err)
	}
	postAgg := NodeToDruidPostAgg("cpm", node)
	m, ok := postAgg["fields"].([]interface{})
	if !ok || len(m) != 2 {
		t.Errorf("NodeToDruidPostAgg: unexpected fields structure: %v", postAgg)
	}
	// Check individual field post-aggregations
	for _, field := range m {
		f, ok := field.(map[string]interface{})
		if !ok {
			t.Errorf("NodeToDruidPostAgg: unexpected field type: %v", field)
			continue
		}
		if f["type"] != "field" {
			t.Errorf("NodeToDruidPostAgg: unexpected field type: %v", f["type"])
		}
	}
}
