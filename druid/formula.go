package druid

import (
	"errors"
	"strconv"
	"unicode"
)

// Noeud d’arbre d’expression
type FormulaNode struct {
	Op    string
	Left  *FormulaNode
	Right *FormulaNode
	Value string
}

// Tokenization de la formule
func tokenizeFormula(expr string) ([]string, error) {
	tokens := []string{}
	i := 0
	for i < len(expr) {
		c := expr[i]
		if unicode.IsSpace(rune(c)) {
			i++
			continue
		}
		if c == '(' || c == ')' || c == '+' || c == '-' || c == '*' || c == '/' {
			tokens = append(tokens, string(c))
			i++
			continue
		}
		start := i
		for i < len(expr) && (unicode.IsLetter(rune(expr[i])) || unicode.IsDigit(rune(expr[i])) || expr[i] == '.' || expr[i] == '_') {
			i++
		}
		if start != i {
			tokens = append(tokens, expr[start:i])
		} else {
			return nil, errors.New("formula: invalid token at " + expr[i:])
		}
	}
	return tokens, nil
}

// Parseur récursif (arithmétique standard)
func parseFormulaExpr(tokens []string) (*FormulaNode, []string, error) {
	node, tokens, err := parseTerm(tokens)
	if err != nil {
		return nil, tokens, err
	}
	for len(tokens) > 0 && (tokens[0] == "+" || tokens[0] == "-") {
		op := tokens[0]
		right, tokens2, err := parseTerm(tokens[1:])
		if err != nil {
			return nil, tokens, err
		}
		node = &FormulaNode{Op: op, Left: node, Right: right}
		tokens = tokens2
	}
	return node, tokens, nil
}

func parseTerm(tokens []string) (*FormulaNode, []string, error) {
	node, tokens, err := parseFactor(tokens)
	if err != nil {
		return nil, tokens, err
	}
	for len(tokens) > 0 && (tokens[0] == "*" || tokens[0] == "/") {
		op := tokens[0]
		right, tokens2, err := parseFactor(tokens[1:])
		if err != nil {
			return nil, tokens, err
		}
		node = &FormulaNode{Op: op, Left: node, Right: right}
		tokens = tokens2
	}
	return node, tokens, nil
}

func parseFactor(tokens []string) (*FormulaNode, []string, error) {
	if len(tokens) == 0 {
		return nil, tokens, errors.New("unexpected end of formula")
	}
	if tokens[0] == "(" {
		node, tokens2, err := parseFormulaExpr(tokens[1:])
		if err != nil {
			return nil, tokens, err
		}
		if len(tokens2) == 0 || tokens2[0] != ")" {
			return nil, tokens, errors.New("missing )")
		}
		return node, tokens2[1:], nil
	}
	tok := tokens[0]
	// Ajout : gestion des fonctions sum(x)
	if len(tokens) > 2 && tokens[1] == "(" {
		fnName := tok
		argNode, rest, err := parseFormulaExpr(tokens[2:])
		if err != nil {
			return nil, tokens, err
		}
		if len(rest) == 0 || rest[0] != ")" {
			return nil, tokens, errors.New("missing ) after function argument")
		}
		return &FormulaNode{Op: "func", Value: fnName, Left: argNode}, rest[1:], nil
	}
	if _, err := strconv.ParseFloat(tok, 64); err == nil {
		return &FormulaNode{Value: tok}, tokens[1:], nil
	}
	return &FormulaNode{Value: tok}, tokens[1:], nil
}

// Parse une formule "1000 * revenue / impressions" en arbre d'expression
func ParseFormula(formula string) (*FormulaNode, error) {
	tokens, err := tokenizeFormula(formula)
	if err != nil {
		return nil, err
	}
	node, rest, err := parseFormulaExpr(tokens)
	if err != nil {
		return nil, err
	}
	if len(rest) != 0 {
		return nil, errors.New("trailing tokens in formula")
	}
	return node, nil
}

// Récupère tous les "leafs" (champs nécessaires pour la formule)
func CollectLeafFields(node *FormulaNode) []string {
	if node == nil {
		return nil
	}
	if node.Op == "" {
		if _, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return nil
		}
		return []string{node.Value}
	}
	l := CollectLeafFields(node.Left)
	r := CollectLeafFields(node.Right)
	return append(l, r...)
}

// Convertit l'arbre de formule en postAggregation Druid (map[string]interface{})
func NodeToDruidPostAgg(name string, node *FormulaNode) map[string]interface{} {
	if node.Op == "" {
		if _, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return map[string]interface{}{
				"type":  "constant",
				"value": mustParseFloat(node.Value),
			}
		}
		return map[string]interface{}{
			"type":      "fieldAccess",
			"fieldName": node.Value,
		}
	}
	if node.Op == "func" && node.Value == "sum" {
		// On suppose que l'agg Druid s'appelle "sum_<champ>"
		return map[string]interface{}{
			"type":      "fieldAccess",
			"fieldName": "sum_" + node.Left.Value,
		}
	}
	fnMap := map[string]string{
		"+": "+", "-": "-", "*": "*", "/": "/",
	}
	return map[string]interface{}{
		"type": "arithmetic",
		"name": name,
		"fn":   fnMap[node.Op],
		"fields": []interface{}{
			NodeToDruidPostAgg("", node.Left),
			NodeToDruidPostAgg("", node.Right),
		},
	}
}

func mustParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
