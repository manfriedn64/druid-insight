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
	typ, ok := postAgg["type"].(string)
	if !ok {
		t.Fatalf("NodeToDruidPostAgg: missing type field")
	}
	if typ == "arithmetic" {
		fields, ok := postAgg["fields"].([]interface{})
		if !ok || len(fields) != 2 {
			t.Errorf("NodeToDruidPostAgg: expected 2 fields for arithmetic, got %v", postAgg["fields"])
		}
	} else if typ == "fieldAccess" {
		// OK, nothing to check
	} else {
		t.Errorf("NodeToDruidPostAgg: unexpected type %v", typ)
	}
}

func TestParseFormula_SumFunction(t *testing.T) {
	node, err := ParseFormula("sum(revenue) / sum(imps)")
	if err != nil {
		t.Fatalf("ParseFormula failed: %v", err)
	}
	if node.Op != "/" {
		t.Errorf("Expected root op '/', got %q", node.Op)
	}
	if node.Left.Op != "func" || node.Left.Value != "sum" {
		t.Errorf("Expected left func 'sum', got %+v", node.Left)
	}
	if node.Right.Op != "func" || node.Right.Value != "sum" {
		t.Errorf("Expected right func 'sum', got %+v", node.Right)
	}
	postAgg := NodeToDruidPostAgg("cpm", node)
	left := postAgg["fields"].([]interface{})[0].(map[string]interface{})
	right := postAgg["fields"].([]interface{})[1].(map[string]interface{})
	if left["fieldName"] != "sum_revenue" {
		t.Errorf("Expected left fieldName 'sum_revenue', got %v", left["fieldName"])
	}
	if right["fieldName"] != "sum_imps" {
		t.Errorf("Expected right fieldName 'sum_imps', got %v", right["fieldName"])
	}
}
