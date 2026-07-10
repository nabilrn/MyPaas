package dbstudio

import (
	"fmt"
	"sort"
	"strings"

	"mypaas/internal/errs"
)

const enumValueSeparator = "\x1f"
const maxRowSearchLength = 120
const maxEnumFilters = 12

type enumFilter struct {
	Column Column
	Value  string
}

func normalizeRowSearch(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > maxRowSearchLength {
		return "", fmt.Errorf("%w: row search must be %d characters or fewer", errs.ErrValidation, maxRowSearchLength)
	}
	return value, nil
}

func enumFilters(filters map[string]string, columns []Column) ([]enumFilter, error) {
	if len(filters) == 0 {
		return nil, nil
	}
	if len(filters) > maxEnumFilters {
		return nil, fmt.Errorf("%w: too many enum filters", errs.ErrValidation)
	}

	byName := columnByName(columns)
	names := make([]string, 0, len(filters))
	for name := range filters {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]enumFilter, 0, len(names))
	for _, name := range names {
		value := strings.TrimSpace(filters[name])
		if value == "" {
			continue
		}
		column, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("%w: filter column %q is not available", errs.ErrValidation, name)
		}
		if len(column.EnumValues) == 0 {
			return nil, fmt.Errorf("%w: filter column %q is not an enum", errs.ErrValidation, name)
		}
		if !containsString(column.EnumValues, value) {
			return nil, fmt.Errorf("%w: filter value is not valid for enum column %q", errs.ErrValidation, name)
		}
		out = append(out, enumFilter{Column: column, Value: value})
	}
	return out, nil
}

func searchableColumns(columns []Column) []Column {
	out := make([]Column, 0, len(columns))
	for _, column := range columns {
		if isSearchableColumn(column) {
			out = append(out, column)
		}
	}
	return out
}

func isSearchableColumn(column Column) bool {
	dataType := strings.ToLower(column.DataType)
	switch {
	case strings.Contains(dataType, "blob"),
		strings.Contains(dataType, "binary"),
		strings.Contains(dataType, "bytea"),
		strings.Contains(dataType, "json"),
		strings.Contains(dataType, "geometry"),
		strings.Contains(dataType, "geography"),
		strings.Contains(dataType, "xml"):
		return false
	default:
		return true
	}
}

func likePattern(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return "%" + value + "%"
}

func parseEnumValues(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(strings.ToLower(value), "enum(") {
		return parseMySQLEnumValues(value)
	}

	parts := strings.Split(value, enumValueSeparator)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseMySQLEnumValues(value string) []string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(value), "enum(") || !strings.HasSuffix(value, ")") {
		return nil
	}
	body := value[len("enum(") : len(value)-1]
	out := make([]string, 0)
	var current strings.Builder
	inQuote := false
	for index := 0; index < len(body); index++ {
		ch := body[index]
		switch {
		case !inQuote && ch == '\'':
			inQuote = true
		case inQuote && ch == '\\' && index+1 < len(body):
			index++
			current.WriteByte(body[index])
		case inQuote && ch == '\'' && index+1 < len(body) && body[index+1] == '\'':
			index++
			current.WriteByte('\'')
		case inQuote && ch == '\'':
			inQuote = false
			out = append(out, current.String())
			current.Reset()
		case inQuote:
			current.WriteByte(ch)
		}
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
