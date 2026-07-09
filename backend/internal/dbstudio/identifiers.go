package dbstudio

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"mypaas/internal/errs"
)

const maxRowsLimit = 500

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > maxRowsLimit {
		return maxRowsLimit
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func validateUserSchema(schema string) error {
	if schema == "" || isSystemSchema(schema) {
		return fmt.Errorf("%w: schema is not available for DB Studio", errs.ErrValidation)
	}
	return nil
}

func isSystemSchema(schema string) bool {
	switch strings.ToLower(strings.TrimSpace(schema)) {
	case "information_schema", "pg_catalog", "pg_toast", "mysql", "performance_schema", "sys":
		return true
	default:
		return false
	}
}

func quotePostgresIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteMySQLIdent(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func scanRows(rows *sql.Rows, columns []Column, limit, offset int) (RowPage, error) {
	names, err := rows.Columns()
	if err != nil {
		return RowPage{}, err
	}
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		values, ptrs := scanTargets(len(names))
		if err := rows.Scan(ptrs...); err != nil {
			return RowPage{}, err
		}
		out = append(out, rowMap(names, values))
	}
	if err := rows.Err(); err != nil {
		return RowPage{}, err
	}
	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	return RowPage{Columns: columns, Rows: out, Limit: limit, Offset: offset, HasMore: hasMore}, nil
}

func scanTargets(count int) ([]any, []any) {
	values := make([]any, count)
	ptrs := make([]any, count)
	for index := range values {
		ptrs[index] = &values[index]
	}
	return values, ptrs
}

func rowMap(names []string, values []any) map[string]any {
	row := make(map[string]any, len(names))
	for index, name := range names {
		row[name] = jsonValue(values[index])
	}
	return row
}

func jsonValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	case time.Time:
		return typed.Format(time.RFC3339Nano)
	default:
		return typed
	}
}

func columnByName(columns []Column) map[string]Column {
	out := make(map[string]Column, len(columns))
	for _, column := range columns {
		out[column.Name] = column
	}
	return out
}

func primaryKeyColumns(columns []Column) []Column {
	out := make([]Column, 0)
	for _, column := range columns {
		if column.PrimaryKey {
			out = append(out, column)
		}
	}
	return out
}
