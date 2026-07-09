package dbstudio

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"mypaas/internal/errs"
)

type mysqlAdapter struct{}

func (mysqlAdapter) Schemas(ctx context.Context, conn *sql.DB) ([]Schema, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
ORDER BY schema_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSchemas(rows)
}

func (mysqlAdapter) Tables(ctx context.Context, conn *sql.DB, schema string) ([]Table, error) {
	if err := validateUserSchema(schema); err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, `
SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_schema = ?
  AND table_type = 'BASE TABLE'
ORDER BY table_name`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTables(rows)
}

func (a mysqlAdapter) Columns(ctx context.Context, conn *sql.DB, schema, table string) ([]Column, error) {
	if err := validateTable(ctx, conn, a, schema, table); err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, `
SELECT column_name, column_type, is_nullable = 'YES',
       column_key = 'PRI',
       extra LIKE '%auto_increment%' OR generation_expression <> ''
FROM information_schema.columns
WHERE table_schema = ?
  AND table_name = ?
ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanColumns(rows)
}

func (a mysqlAdapter) Rows(ctx context.Context, conn *sql.DB, query RowQuery) (RowPage, error) {
	columns, err := a.Columns(ctx, conn, query.Schema, query.Table)
	if err != nil {
		return RowPage{}, err
	}
	limit := normalizeLimit(query.Limit)
	sqlText := mysqlSelectSQL(query.Schema, query.Table, columns)
	rows, err := conn.QueryContext(ctx, sqlText, limit+1, normalizeOffset(query.Offset))
	if err != nil {
		return RowPage{}, err
	}
	defer rows.Close()
	return scanRows(rows, columns, limit, normalizeOffset(query.Offset))
}

func (a mysqlAdapter) Insert(ctx context.Context, conn *sql.DB, mutation Mutation) error {
	columns, err := a.Columns(ctx, conn, mutation.Schema, mutation.Table)
	if err != nil {
		return err
	}
	sqlText, args, err := mysqlInsertSQL(mutation, columns)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, sqlText, args...)
	return err
}

func (a mysqlAdapter) Update(ctx context.Context, conn *sql.DB, mutation Mutation) error {
	columns, err := a.Columns(ctx, conn, mutation.Schema, mutation.Table)
	if err != nil {
		return err
	}
	sqlText, args, err := mysqlUpdateSQL(mutation, columns)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, sqlText, args...)
	return err
}

func (a mysqlAdapter) Delete(ctx context.Context, conn *sql.DB, mutation Mutation) error {
	columns, err := a.Columns(ctx, conn, mutation.Schema, mutation.Table)
	if err != nil {
		return err
	}
	sqlText, args, err := mysqlDeleteSQL(mutation, columns)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, sqlText, args...)
	return err
}

func mysqlSelectSQL(schema, table string, columns []Column) string {
	names := quotedColumnList(columns, quoteMySQLIdent)
	order := mysqlOrderClause(columns)
	return fmt.Sprintf("SELECT %s FROM %s.%s%s LIMIT ? OFFSET ?",
		names, quoteMySQLIdent(schema), quoteMySQLIdent(table), order)
}

func mysqlOrderClause(columns []Column) string {
	pks := primaryKeyColumns(columns)
	if len(pks) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pks))
	for _, column := range pks {
		parts = append(parts, quoteMySQLIdent(column.Name))
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}

func mysqlInsertSQL(m Mutation, columns []Column) (string, []any, error) {
	names, values, err := mutationValues(m.Values, columns)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)",
		quoteMySQLIdent(m.Schema), quoteMySQLIdent(m.Table),
		joinQuoted(names, quoteMySQLIdent), mysqlPlaceholders(len(names))), values, nil
}

func mysqlUpdateSQL(m Mutation, columns []Column) (string, []any, error) {
	names, values, err := mutationValues(m.Values, columns)
	if err != nil {
		return "", nil, err
	}
	where, pkValues, err := mysqlPrimaryKeyWhere(m.PrimaryKey, columns)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("UPDATE %s.%s SET %s WHERE %s",
		quoteMySQLIdent(m.Schema), quoteMySQLIdent(m.Table),
		mysqlSetList(names), where), append(values, pkValues...), nil
}

func mysqlDeleteSQL(m Mutation, columns []Column) (string, []any, error) {
	where, values, err := mysqlPrimaryKeyWhere(m.PrimaryKey, columns)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("DELETE FROM %s.%s WHERE %s",
		quoteMySQLIdent(m.Schema), quoteMySQLIdent(m.Table), where), values, nil
}

func mysqlSetList(names []string) string {
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, quoteMySQLIdent(name)+" = ?")
	}
	return strings.Join(parts, ", ")
}

func mysqlPrimaryKeyWhere(values map[string]any, columns []Column) (string, []any, error) {
	pks := primaryKeyColumns(columns)
	if len(pks) == 0 {
		return "", nil, fmt.Errorf("%w: table needs a primary key for write actions", errs.ErrValidation)
	}
	args := make([]any, 0, len(pks))
	parts := make([]string, 0, len(pks))
	for _, column := range pks {
		value, ok := values[column.Name]
		if !ok {
			return "", nil, fmt.Errorf("%w: primary key %q is required", errs.ErrValidation, column.Name)
		}
		args = append(args, value)
		parts = append(parts, quoteMySQLIdent(column.Name)+" = ?")
	}
	return strings.Join(parts, " AND "), args, nil
}

func mysqlPlaceholders(count int) string {
	parts := make([]string, 0, count)
	for index := 0; index < count; index++ {
		parts = append(parts, "?")
	}
	return strings.Join(parts, ", ")
}
