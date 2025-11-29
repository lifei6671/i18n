package i18n

import (
	"fmt"
	"strings"
	"sync"
)

///////////////////////////////////////////////////////////////////////////////
// FORMATTER REGISTRY
///////////////////////////////////////////////////////////////////////////////

var formatterRegistry = map[string]FormatterFunc{}
var regMutex sync.RWMutex

// FormatterFunc represents the user-defined or built-in formatter.
type FormatterFunc func(input any, arg string) (any, error)

// RegisterFormatter allows user to register custom formatters.
func RegisterFormatter(name string, f FormatterFunc) {
	regMutex.Lock()
	defer regMutex.Unlock()
	formatterRegistry[name] = f
}

// applyRegisteredFormatter applies a formatter by name.
func applyRegisteredFormatter(v any, name, arg string) (any, error) {
	regMutex.RLock()
	f, ok := formatterRegistry[name]
	regMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown formatter: %s", name)
	}
	return f(v, arg)
}

///////////////////////////////////////////////////////////////////////////////
// DEFAULT FORMATTERS REGISTERED AT INIT
///////////////////////////////////////////////////////////////////////////////

func init() {
	// register built-in formatters
	RegisterFormatter("upper", func(v any, arg string) (any, error) {
		return strings.ToUpper(fmt.Sprint(v)), nil
	})
	RegisterFormatter("lower", func(v any, arg string) (any, error) {
		return strings.ToLower(fmt.Sprint(v)), nil
	})
	RegisterFormatter("title", func(v any, arg string) (any, error) {
		return strings.Title(fmt.Sprint(v)), nil
	})

	RegisterFormatter("number", func(v any, arg string) (any, error) {
		return formatNumber(v, arg)
	})
	RegisterFormatter("currency", func(v any, arg string) (any, error) {
		return formatCurrency(v, arg)
	})
	RegisterFormatter("date", func(v any, arg string) (any, error) {
		return formatDate(v, arg)
	})
}
