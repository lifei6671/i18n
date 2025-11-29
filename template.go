package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

///////////////////////////////////////////////////////////////////////////////
// AST DEFINITIONS
///////////////////////////////////////////////////////////////////////////////

// Node is the interface for all AST nodes.
type Node interface {
	// Eval evaluates the node with given args and returns string output.
	Eval(args map[string]any) (string, error)
}

// TextNode represents a static text segment.
type TextNode struct {
	Text string
}

func (t *TextNode) Eval(_ map[string]any) (string, error) {
	return t.Text, nil
}

// Formatter represents a single formatter in the chain.
type Formatter struct {
	Name string
	Arg  string
}

// Conditional represents a ternary condition chain inside a placeholder.
type Conditional struct {
	Op        string // "eq", "gt", "lt"
	TestValue string
	TrueExpr  string
	FalseExpr string
}

// PlaceholderNode represents: {path | formatter:arg | ...}
type PlaceholderNode struct {
	Path       string
	Formatters []Formatter
	Cond       *Conditional // optional
}

func (p *PlaceholderNode) Eval(args map[string]any) (string, error) {
	// Resolve base value
	value, ok := getValueByPath(args, p.Path)
	if !ok {
		return "", fmt.Errorf("value not found: %s", p.Path)
	}

	var err error
	// Apply chained formatters
	for _, f := range p.Formatters {
		value, err = applyRegisteredFormatter(value, f.Name, f.Arg)
		if err != nil {
			return "", err
		}
	}

	// Conditional operator
	if p.Cond != nil {
		ok, err := compareValues(value, p.Cond.Op, p.Cond.TestValue)
		if err != nil {
			return "", err
		}
		if ok {
			return RenderTemplate(p.Cond.TrueExpr, args)
		}
		return RenderTemplate(p.Cond.FalseExpr, args)
	}

	return fmt.Sprint(value), nil
}

// TemplateAST is a whole parsed template.
type TemplateAST []Node

func (t TemplateAST) Eval(args map[string]any) (string, error) {
	var buf bytes.Buffer
	for _, node := range t {
		s, err := node.Eval(args)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
	}
	return buf.String(), nil
}

///////////////////////////////////////////////////////////////////////////////
// AST CACHE
///////////////////////////////////////////////////////////////////////////////

var (
	astCache   = map[string]TemplateAST{}
	cacheMutex sync.RWMutex
)

// RenderTemplate is now AST-powered with caching.
//
// If the template has been parsed once, parsing is skipped and the cached AST is used.
func RenderTemplate(tpl string, args map[string]any) (string, error) {
	// Fast path: get cached AST
	cacheMutex.RLock()
	ast, ok := astCache[tpl]
	cacheMutex.RUnlock()

	if !ok {
		// Parse and cache
		var err error
		ast, err = ParseTemplate(tpl)
		if err != nil {
			return tpl, err
		}
		cacheMutex.Lock()
		astCache[tpl] = ast
		cacheMutex.Unlock()
	}

	// Execute AST
	return ast.Eval(args)
}

///////////////////////////////////////////////////////////////////////////////
// TEMPLATE PARSER
///////////////////////////////////////////////////////////////////////////////

// ParseTemplate parses tpl string into an AST (TemplateAST).
// Runtime version: supports nested `{}` inside a placeholder,
// and is tolerant to unmatched '{' – unclosed '{' will be treated as plain text.
func ParseTemplate(tpl string) (TemplateAST, error) {
	runes := []rune(tpl)
	n := len(runes)

	var nodes TemplateAST
	var buf bytes.Buffer

	i := 0
	for i < n {
		// 普通字符，累积到文本缓冲
		if runes[i] != '{' {
			buf.WriteRune(runes[i])
			i++
			continue
		}

		// 遇到 '{'，先 flush 文本节点
		if buf.Len() > 0 {
			nodes = append(nodes, &TextNode{Text: buf.String()})
			buf.Reset()
		}

		// 尝试解析一个占位符，支持嵌套花括号
		start := i
		depth := 1
		j := i + 1

		for j < n && depth > 0 {
			switch runes[j] {
			case '{':
				depth++
			case '}':
				depth--
			}
			j++
		}

		if depth != 0 {
			// 没有找到配对的 '}'，宽松模式：把这个 '{' 当普通字符输出
			buf.WriteRune(runes[start])
			i = start + 1
			continue
		}

		// 此时 j 指向的是“匹配的那个 '}' 的下一个位置”
		raw := string(runes[start+1 : j-1])
		i = j // 继续处理后面的内容

		ph, err := parsePlaceholder(raw)
		if err != nil {
			// 占位符内部语法有问题，宽松模式：原样输出
			buf.WriteString("{" + raw + "}")
			continue
		}

		// 把当前累积的文本节点 flush（一般为空，但为了稳妥）
		if buf.Len() > 0 {
			nodes = append(nodes, &TextNode{Text: buf.String()})
			buf.Reset()
		}

		nodes = append(nodes, ph)
	}

	// 收尾文本
	if buf.Len() > 0 {
		nodes = append(nodes, &TextNode{Text: buf.String()})
	}

	return nodes, nil
}

///////////////////////////////////////////////////////////////////////////////
// PLACEHOLDER PARSER
///////////////////////////////////////////////////////////////////////////////

// parsePlaceholder parses the expression inside `{ ... }`.
func parsePlaceholder(expr string) (*PlaceholderNode, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, errors.New("empty placeholder expression")
	}

	parts := strings.Split(expr, "|")
	if len(parts) == 0 {
		return nil, errors.New("empty placeholder")
	}

	ph := &PlaceholderNode{
		Path: strings.TrimSpace(parts[0]),
	}

	for i := 1; i < len(parts); i++ {
		seg := strings.TrimSpace(parts[i])
		if seg == "" {
			return nil, fmt.Errorf("empty formatter segment")
		}

		// conditional
		if strings.Contains(seg, "?") {
			cond, err := parseConditional(seg)
			if err != nil {
				return nil, err
			}
			ph.Cond = cond
			continue
		}

		name, arg := parseFormatterSegment(seg)
		if name == "" {
			return nil, fmt.Errorf("empty formatter name in segment %q", seg)
		}
		ph.Formatters = append(ph.Formatters, Formatter{
			Name: name,
			Arg:  arg,
		})
	}

	return ph, nil
}

// parseFormatterSegment parses "number:2" etc.
func parseFormatterSegment(seg string) (name, arg string) {
	ff := strings.SplitN(seg, ":", 2)
	name = strings.TrimSpace(ff[0])
	if len(ff) > 1 {
		arg = strings.TrimSpace(ff[1])
	}
	return
}

// parseConditional parses "eq:0?A:B".
func parseConditional(expr string) (*Conditional, error) {
	q := strings.SplitN(expr, "?", 2)
	if len(q) != 2 {
		return nil, fmt.Errorf("invalid conditional: %s", expr)
	}
	condPart := q[0]
	trueFalse := q[1]
	tf := strings.SplitN(trueFalse, ":", 2)
	if len(tf) != 2 {
		return nil, fmt.Errorf("invalid conditional: %s", expr)
	}

	condKV := strings.SplitN(condPart, ":", 2)
	if len(condKV) != 2 {
		return nil, fmt.Errorf("invalid condition: %s", condPart)
	}

	return &Conditional{
		Op:        strings.TrimSpace(condKV[0]),
		TestValue: strings.TrimSpace(condKV[1]),
		TrueExpr:  strings.TrimSpace(tf[0]),
		FalseExpr: strings.TrimSpace(tf[1]),
	}, nil
}

///////////////////////////////////////////////////////////////////////////////
// REMAINS: VALUE RESOLUTION / NUMBER / DATE (reuse your existing logic)
///////////////////////////////////////////////////////////////////////////////

func getValueByPath(args map[string]any, path string) (any, bool) {
	segs := strings.Split(path, ".")
	var current any = args

	for _, seg := range segs {
		log.Println(seg)
		switch c := current.(type) {
		case map[string]any:
			v, ok := c[seg]
			if !ok {
				return nil, false
			}
			current = v
		default:
			r := reflect.ValueOf(c)
			if r.Kind() == reflect.Ptr {
				r = r.Elem()
			}
			if r.Kind() != reflect.Struct {
				return nil, false
			}
			f := r.FieldByNameFunc(func(name string) bool {
				return strings.EqualFold(name, seg)
			})
			if !f.IsValid() {
				return nil, false
			}
			current = f.Interface()
		}
	}
	return current, true
}

func formatDate(v any, layout string) (string, error) {
	if layout == "" {
		layout = "2006-01-02"
	}
	switch t := v.(type) {
	case time.Time:
		return t.Format(layout), nil
	case *time.Time:
		return t.Format(layout), nil
	case string:
		tt, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return "", err
		}
		return tt.Format(layout), nil
	default:
		return "", fmt.Errorf("not a time: %v", v)
	}
}
func formatNumber(v any, precision string) (string, error) {
	var f float64

	switch n := v.(type) {
	case int:
		f = float64(n)
	case int64:
		f = float64(n)
	case float32:
		f = float64(n)
	case float64:
		f = n

	case string:
		n = strings.TrimSpace(n)
		n = strings.ReplaceAll(n, ",", "")
		ff, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return "", fmt.Errorf("number formatter: cannot parse %q", n)
		}
		f = ff

	default:
		return "", fmt.Errorf("number formatter requires numeric or numeric-string type, got %T", v)
	}

	p := 0
	if precision != "" {
		pi, err := strconv.Atoi(precision)
		if err != nil {
			return "", fmt.Errorf("number formatter: invalid precision %q", precision)
		}
		p = pi
	}

	s := fmt.Sprintf("%."+strconv.Itoa(p)+"f", f)
	return addThousandsSep(s), nil
}

func addThousandsSep(s string) string {
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	parts := strings.Split(s, ".")
	intPart := parts[0]

	var buf bytes.Buffer
	for i, c := range intPart {
		if i != 0 && (len(intPart)-i)%3 == 0 {
			buf.WriteRune(',')
		}
		buf.WriteRune(c)
	}

	if len(parts) > 1 {
		buf.WriteRune('.')
		buf.WriteString(parts[1])
	}

	if neg {
		return "-" + buf.String()
	}
	return buf.String()
}
func formatCurrency(v any, arg string) (string, error) {
	symbol := "$"
	if arg != "" {
		symbol = arg
	}

	var f float64
	switch n := v.(type) {
	case int:
		f = float64(n)
	case int64:
		f = float64(n)
	case float32:
		f = float64(n)
	case float64:
		f = n

	case string:
		// strip currency symbols
		n = strings.TrimSpace(n)
		n = strings.TrimPrefix(n, "$")
		n = strings.TrimPrefix(n, "¥")
		n = strings.TrimPrefix(n, "€")
		n = strings.TrimPrefix(n, "£")
		n = strings.TrimPrefix(n, symbol)

		// remove thousand separators
		n = strings.ReplaceAll(n, ",", "")

		// parse
		ff, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return "", fmt.Errorf("currency formatter: cannot parse number from %q", v)
		}
		f = ff

	default:
		return "", fmt.Errorf("currency formatter requires numeric or numeric-string type, got %T", v)
	}

	// format with thousand separator
	s := fmt.Sprintf("%.2f", f)
	s = addThousandsSep(s)

	return symbol + s, nil
}

func compareValues(v any, op string, test string) (bool, error) {
	switch vv := v.(type) {
	case int, int64, float64, float32:
		return compareNumbers(vv, op, test)
	case string:
		switch op {
		case "eq":
			return vv == test, nil
		default:
			return false, fmt.Errorf("unsupported string op: %s", op)
		}
	default:
		return false, fmt.Errorf("unsupported type for compare: %T", v)
	}
}

func compareNumbers(v any, op, test string) (bool, error) {
	var lv float64
	switch t := v.(type) {
	case int:
		lv = float64(t)
	case int64:
		lv = float64(t)
	case float64:
		lv = t
	case float32:
		lv = float64(t)
	}
	rv, err := strconv.ParseFloat(test, 64)
	if err != nil {
		return false, err
	}

	switch op {
	case "eq":
		return lv == rv, nil
	case "gt":
		return lv > rv, nil
	case "lt":
		return lv < rv, nil
	default:
		return false, fmt.Errorf("unknown op: %s", op)
	}
}

// ValidateTemplate does a strict validation for linting purpose:
//  1. checks brace balance
//  2. parses into AST
//  3. checks formatter existence and basic arguments
func ValidateTemplate(tpl string) error {
	// 1. 括号配对检查
	if err := checkBraces(tpl); err != nil {
		return err
	}

	// 2. 解析 AST（宽松版 ParseTemplate 在这一步不会再因 unclosed '{' 报错）
	ast, err := ParseTemplate(tpl)
	if err != nil {
		return err
	}

	// 3. 对 AST 做 formatter / 条件 / 参数校验
	for _, node := range ast {
		ph, ok := node.(*PlaceholderNode)
		if !ok {
			continue
		}

		if strings.TrimSpace(ph.Path) == "" {
			return fmt.Errorf("placeholder has empty path")
		}

		// 校验 formatter 是否注册，以及基础参数
		for _, f := range ph.Formatters {
			name := strings.TrimSpace(f.Name)
			if name == "" {
				return fmt.Errorf("empty formatter name")
			}
			regMutex.RLock()
			_, exists := formatterRegistry[name]
			regMutex.RUnlock()
			if !exists {
				return fmt.Errorf("unknown formatter: %s", name)
			}

			switch name {
			case "number":
				if f.Arg != "" {
					if _, err := strconv.Atoi(f.Arg); err != nil {
						return fmt.Errorf("invalid precision for number formatter: %q", f.Arg)
					}
				}
			}
		}

		// 条件表达式检查
		if ph.Cond != nil {
			switch ph.Cond.Op {
			case "eq", "gt", "lt":
				// ok
			default:
				return fmt.Errorf("unknown conditional operator: %s", ph.Cond.Op)
			}
			if strings.TrimSpace(ph.Cond.TrueExpr) == "" || strings.TrimSpace(ph.Cond.FalseExpr) == "" {
				return fmt.Errorf("invalid conditional expression: true/false branch must not be empty")
			}
		}
	}

	return nil
}

// checkBraces checks that all '{' and '}' are balanced at the template level.
func checkBraces(tpl string) error {
	runes := []rune(tpl)
	depth := 0
	firstOpen := -1

	for i, r := range runes {
		switch r {
		case '{':
			if depth == 0 {
				firstOpen = i
			}
			depth++
		case '}':
			if depth == 0 {
				return fmt.Errorf("extra closing '}' at position %d", i)
			}
			depth--
		}
	}

	if depth != 0 && firstOpen >= 0 {
		return fmt.Errorf("unclosed placeholder starting at position %d", firstOpen)
	}
	return nil
}
