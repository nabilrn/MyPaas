package dbstudio

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"mypaas/internal/errs"
)

type postgresAdapter struct{}

func (postgresAdapter) Schemas(ctx context.Context, conn *sql.DB) ([]Schema, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
  AND schema_name NOT LIKE 'pg_temp_%'
  AND schema_name NOT LIKE 'pg_toast_temp_%'
ORDER BY schema_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSchemas(rows)
}

func (postgresAdapter) Tables(ctx context.Context, conn *sql.DB, schema string) ([]Table, error) {
	if err := validateUserSchema(schema); err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, `
SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_schema = $1
  AND table_type = 'BASE TABLE'
ORDER BY table_name`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTables(rows)
}

func (a postgresAdapter) Columns(ctx context.Context, conn *sql.DB, schema, table string) ([]Column, error) {
	if err := validateTable(ctx, conn, a, schema, table); err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, `
SELECT c.column_name, c.data_type, c.is_nullable = 'YES',
       COALESCE(tc.constraint_name IS NOT NULL, false),
       c.column_default IS NOT NULL AND c.column_default <> '',
       COALESCE(enum_values.labels, '')
FROM information_schema.columns c
LEFT JOIN information_schema.key_column_usage k
  ON k.table_schema = c.table_schema
 AND k.table_name = c.table_name
 AND k.column_name = c.column_name
LEFT JOIN information_schema.table_constraints tc
 ON tc.constraint_schema = k.constraint_schema
 AND tc.constraint_name = k.constraint_name
 AND tc.constraint_type = 'PRIMARY KEY'
LEFT JOIN LATERAL (
  SELECT string_agg(e.enumlabel, $3::text ORDER BY e.enumsortorder) AS labels
  FROM pg_catalog.pg_type t
  JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
  JOIN pg_catalog.pg_enum e ON e.enumtypid = t.oid
  WHERE n.nspname = c.udt_schema
    AND t.typname = c.udt_name
) enum_values ON true
WHERE c.table_schema = $1
  AND c.table_name = $2
ORDER BY c.ordinal_position`, schema, table, enumValueSeparator)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanColumns(rows)
}

func (a postgresAdapter) Rows(ctx context.Context, conn *sql.DB, query RowQuery) (RowPage, error) {
	columns, err := a.Columns(ctx, conn, query.Schema, query.Table)
	if err != nil {
		return RowPage{}, err
	}
	limit := normalizeLimit(query.Limit)
	offset := normalizeOffset(query.Offset)
	sqlText, args, err := postgresSelectSQL(query.Schema, query.Table, columns, query, limit, offset)
	if err != nil {
		return RowPage{}, err
	}
	rows, err := conn.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return RowPage{}, err
	}
	defer rows.Close()
	return scanRows(rows, columns, limit, offset)
}

func (a postgresAdapter) Insert(ctx context.Context, conn *sql.DB, mutation Mutation) error {
	columns, err := a.Columns(ctx, conn, mutation.Schema, mutation.Table)
	if err != nil {
		return err
	}
	sqlText, args, err := postgresInsertSQL(mutation, columns)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, sqlText, args...)
	return err
}

func (a postgresAdapter) Update(ctx context.Context, conn *sql.DB, mutation Mutation) error {
	columns, err := a.Columns(ctx, conn, mutation.Schema, mutation.Table)
	if err != nil {
		return err
	}
	sqlText, args, err := postgresUpdateSQL(mutation, columns)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, sqlText, args...)
	return err
}

func (a postgresAdapter) Delete(ctx context.Context, conn *sql.DB, mutation Mutation) error {
	columns, err := a.Columns(ctx, conn, mutation.Schema, mutation.Table)
	if err != nil {
		return err
	}
	sqlText, args, err := postgresDeleteSQL(mutation, columns)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, sqlText, args...)
	return err
}

func postgresSelectSQL(schema, table string, columns []Column, query RowQuery, limit, offset int) (string, []any, error) {
	names := quotedColumnList(columns, quotePostgresIdent)
	where, args, err := postgresWhereClause(columns, query)
	if err != nil {
		return "", nil, err
	}
	order := postgresOrderClause(columns)
	args = append(args, limit+1, offset)
	return fmt.Sprintf("SELECT %s FROM %s.%s%s%s LIMIT $%d OFFSET $%d",
		names, quotePostgresIdent(schema), quotePostgresIdent(table), where, order, len(args)-1, len(args)), args, nil
}

func postgresWhereClause(columns []Column, query RowQuery) (string, []any, error) {
	clauses := make([]string, 0)
	args := make([]any, 0)

	filters, err := enumFilters(query.Filters, columns)
	if err != nil {
		return "", nil, err
	}
	for _, filter := range filters {
		args = append(args, filter.Value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", quotePostgresIdent(filter.Column.Name), len(args)))
	}

	search, err := normalizeRowSearch(query.Search)
	if err != nil {
		return "", nil, err
	}
	if search != "" {
		searchable := searchableColumns(columns)
		if len(searchable) == 0 {
			clauses = append(clauses, "1 = 0")
		} else {
			pattern := likePattern(search)
			parts := make([]string, 0, len(searchable))
			for _, column := range searchable {
				args = append(args, pattern)
				parts = append(parts, fmt.Sprintf("CAST(%s AS TEXT) ILIKE $%d ESCAPE '\\'", quotePostgresIdent(column.Name), len(args)))
			}
			clauses = append(clauses, "("+strings.Join(parts, " OR ")+")")
		}
	}

	if len(clauses) == 0 {
		return "", args, nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args, nil
}

func postgresOrderClause(columns []Column) string {
	pks := primaryKeyColumns(columns)
	if len(pks) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pks))
	for _, column := range pks {
		parts = append(parts, quotePostgresIdent(column.Name))
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}

func postgresInsertSQL(m Mutation, columns []Column) (string, []any, error) {
	names, values, err := mutationValues(m.Values, columns)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)",
		quotePostgresIdent(m.Schema), quotePostgresIdent(m.Table),
		joinQuoted(names, quotePostgresIdent), postgresPlaceholders(1, len(names))), values, nil
}

func postgresUpdateSQL(m Mutation, columns []Column) (string, []any, error) {
	names, values, err := mutationValues(m.Values, columns)
	if err != nil {
		return "", nil, err
	}
	where, pkValues, err := postgresPrimaryKeyWhere(m.PrimaryKey, columns, len(values)+1)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("UPDATE %s.%s SET %s WHERE %s",
		quotePostgresIdent(m.Schema), quotePostgresIdent(m.Table),
		postgresSetList(names), where), append(values, pkValues...), nil
}

func postgresDeleteSQL(m Mutation, columns []Column) (string, []any, error) {
	where, values, err := postgresPrimaryKeyWhere(m.PrimaryKey, columns, 1)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("DELETE FROM %s.%s WHERE %s",
		quotePostgresIdent(m.Schema), quotePostgresIdent(m.Table), where), values, nil
}

func postgresSetList(names []string) string {
	parts := make([]string, 0, len(names))
	for index, name := range names {
		parts = append(parts, fmt.Sprintf("%s = $%d", quotePostgresIdent(name), index+1))
	}
	return strings.Join(parts, ", ")
}

func postgresPrimaryKeyWhere(values map[string]any, columns []Column, start int) (string, []any, error) {
	pks := primaryKeyColumns(columns)
	if len(pks) == 0 {
		return "", nil, fmt.Errorf("%w: table needs a primary key for write actions", errs.ErrValidation)
	}
	args := make([]any, 0, len(pks))
	parts := make([]string, 0, len(pks))
	for index, column := range pks {
		value, ok := values[column.Name]
		if !ok {
			return "", nil, fmt.Errorf("%w: primary key %q is required", errs.ErrValidation, column.Name)
		}
		args = append(args, value)
		parts = append(parts, fmt.Sprintf("%s = $%d", quotePostgresIdent(column.Name), start+index))
	}
	return strings.Join(parts, " AND "), args, nil
}

func postgresPlaceholders(start, count int) string {
	parts := make([]string, 0, count)
	for index := 0; index < count; index++ {
		parts = append(parts, fmt.Sprintf("$%d", start+index))
	}
	return strings.Join(parts, ", ")
}
