package dbstudio

import (
	"context"
	"database/sql"
	"fmt"
)

func (a postgresAdapter) TableDetails(ctx context.Context, conn *sql.DB, schema, table string) (TableDetails, error) {
	columns, err := a.Columns(ctx, conn, schema, table)
	if err != nil {
		return TableDetails{}, err
	}
	foreignKeys, err := postgresForeignKeys(ctx, conn, schema, table)
	if err != nil {
		return TableDetails{}, err
	}
	indexes, err := postgresIndexes(ctx, conn, schema, table)
	if err != nil {
		return TableDetails{}, err
	}
	constraints, err := postgresConstraints(ctx, conn, schema, table)
	if err != nil {
		return TableDetails{}, err
	}
	return TableDetails{Schema: schema, Name: table, Columns: columns, ForeignKeys: foreignKeys, Indexes: indexes, Constraints: constraints}, nil
}

func (a mysqlAdapter) TableDetails(ctx context.Context, conn *sql.DB, schema, table string) (TableDetails, error) {
	columns, err := a.Columns(ctx, conn, schema, table)
	if err != nil {
		return TableDetails{}, err
	}
	foreignKeys, err := mysqlForeignKeys(ctx, conn, schema, table)
	if err != nil {
		return TableDetails{}, err
	}
	indexes, err := mysqlIndexes(ctx, conn, schema, table)
	if err != nil {
		return TableDetails{}, err
	}
	constraints, err := mysqlConstraints(ctx, conn, schema, table)
	if err != nil {
		return TableDetails{}, err
	}
	return TableDetails{Schema: schema, Name: table, Columns: columns, ForeignKeys: foreignKeys, Indexes: indexes, Constraints: constraints}, nil
}

func postgresForeignKeys(ctx context.Context, conn *sql.DB, schema, table string) ([]ForeignKey, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT con.conname,
       src.attname,
       target_ns.nspname,
       target.relname,
       dst.attname,
       CASE con.confupdtype WHEN 'a' THEN 'NO ACTION' WHEN 'r' THEN 'RESTRICT' WHEN 'c' THEN 'CASCADE' WHEN 'n' THEN 'SET NULL' WHEN 'd' THEN 'SET DEFAULT' ELSE '' END,
       CASE con.confdeltype WHEN 'a' THEN 'NO ACTION' WHEN 'r' THEN 'RESTRICT' WHEN 'c' THEN 'CASCADE' WHEN 'n' THEN 'SET NULL' WHEN 'd' THEN 'SET DEFAULT' ELSE '' END
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class source ON source.oid = con.conrelid
JOIN pg_catalog.pg_namespace source_ns ON source_ns.oid = source.relnamespace
JOIN pg_catalog.pg_class target ON target.oid = con.confrelid
JOIN pg_catalog.pg_namespace target_ns ON target_ns.oid = target.relnamespace
JOIN LATERAL generate_subscripts(con.conkey, 1) positions(pos) ON true
JOIN pg_catalog.pg_attribute src ON src.attrelid = source.oid AND src.attnum = con.conkey[positions.pos]
JOIN pg_catalog.pg_attribute dst ON dst.attrelid = target.oid AND dst.attnum = con.confkey[positions.pos]
WHERE con.contype = 'f'
  AND source_ns.nspname = $1
  AND source.relname = $2
ORDER BY con.conname, positions.pos`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ForeignKey, 0)
	for rows.Next() {
		var item ForeignKey
		if err := rows.Scan(&item.Name, &item.Column, &item.ReferencedSchema, &item.ReferencedTable, &item.ReferencedColumn, &item.OnUpdate, &item.OnDelete); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func postgresIndexes(ctx context.Context, conn *sql.DB, schema, table string) ([]Index, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT idx.relname,
       ix.indisunique,
       ix.indisprimary,
       am.amname,
       COALESCE(att.attname, ''),
       pg_get_indexdef(ix.indexrelid)
FROM pg_catalog.pg_class tbl
JOIN pg_catalog.pg_namespace ns ON ns.oid = tbl.relnamespace
JOIN pg_catalog.pg_index ix ON ix.indrelid = tbl.oid
JOIN pg_catalog.pg_class idx ON idx.oid = ix.indexrelid
JOIN pg_catalog.pg_am am ON am.oid = idx.relam
JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS keys(attnum, ordinality) ON true
LEFT JOIN pg_catalog.pg_attribute att ON att.attrelid = tbl.oid AND att.attnum = keys.attnum
WHERE ns.nspname = $1
  AND tbl.relname = $2
ORDER BY idx.relname, keys.ordinality`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Index, 0)
	positions := make(map[string]int)
	for rows.Next() {
		var name, method, column, definition string
		var unique, primary bool
		if err := rows.Scan(&name, &unique, &primary, &method, &column, &definition); err != nil {
			return nil, err
		}
		position, ok := positions[name]
		if !ok {
			position = len(items)
			positions[name] = position
			items = append(items, Index{Name: name, Unique: unique, Primary: primary, Method: method, Definition: definition, Columns: []string{}})
		}
		if column != "" {
			items[position].Columns = append(items[position].Columns, column)
		}
	}
	return items, rows.Err()
}

func postgresConstraints(ctx context.Context, conn *sql.DB, schema, table string) ([]Constraint, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT con.conname,
       CASE con.contype WHEN 'p' THEN 'PRIMARY KEY' WHEN 'u' THEN 'UNIQUE' WHEN 'f' THEN 'FOREIGN KEY' WHEN 'c' THEN 'CHECK' WHEN 'x' THEN 'EXCLUSION' ELSE con.contype::text END,
       COALESCE(att.attname, ''),
       pg_get_constraintdef(con.oid, true)
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class rel ON rel.oid = con.conrelid
JOIN pg_catalog.pg_namespace ns ON ns.oid = rel.relnamespace
LEFT JOIN LATERAL unnest(con.conkey) WITH ORDINALITY AS keys(attnum, ordinality) ON true
LEFT JOIN pg_catalog.pg_attribute att ON att.attrelid = rel.oid AND att.attnum = keys.attnum
WHERE ns.nspname = $1
  AND rel.relname = $2
ORDER BY con.conname, keys.ordinality`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConstraintRows(rows)
}

func mysqlForeignKeys(ctx context.Context, conn *sql.DB, schema, table string) ([]ForeignKey, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT k.constraint_name,
       k.column_name,
       k.referenced_table_schema,
       k.referenced_table_name,
       k.referenced_column_name,
       COALESCE(r.update_rule, ''),
       COALESCE(r.delete_rule, '')
FROM information_schema.key_column_usage k
LEFT JOIN information_schema.referential_constraints r
  ON r.constraint_schema = k.constraint_schema
 AND r.constraint_name = k.constraint_name
 AND r.table_name = k.table_name
WHERE k.table_schema = ?
  AND k.table_name = ?
  AND k.referenced_table_name IS NOT NULL
ORDER BY k.constraint_name, k.ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ForeignKey, 0)
	for rows.Next() {
		var item ForeignKey
		if err := rows.Scan(&item.Name, &item.Column, &item.ReferencedSchema, &item.ReferencedTable, &item.ReferencedColumn, &item.OnUpdate, &item.OnDelete); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func mysqlIndexes(ctx context.Context, conn *sql.DB, schema, table string) ([]Index, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT index_name,
       non_unique = 0,
       index_name = 'PRIMARY',
       index_type,
       COALESCE(column_name, '')
FROM information_schema.statistics
WHERE table_schema = ?
  AND table_name = ?
ORDER BY index_name, seq_in_index`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Index, 0)
	positions := make(map[string]int)
	for rows.Next() {
		var name, method, column string
		var unique, primary bool
		if err := rows.Scan(&name, &unique, &primary, &method, &column); err != nil {
			return nil, err
		}
		position, ok := positions[name]
		if !ok {
			position = len(items)
			positions[name] = position
			items = append(items, Index{Name: name, Unique: unique, Primary: primary, Method: method, Columns: []string{}})
		}
		if column != "" {
			items[position].Columns = append(items[position].Columns, column)
		}
	}
	return items, rows.Err()
}

func mysqlConstraints(ctx context.Context, conn *sql.DB, schema, table string) ([]Constraint, error) {
	rows, err := conn.QueryContext(ctx, `
SELECT tc.constraint_name,
       tc.constraint_type,
       COALESCE(k.column_name, ''),
       COALESCE(cc.check_clause, '')
FROM information_schema.table_constraints tc
LEFT JOIN information_schema.key_column_usage k
  ON k.constraint_schema = tc.constraint_schema
 AND k.table_schema = tc.table_schema
 AND k.table_name = tc.table_name
 AND k.constraint_name = tc.constraint_name
LEFT JOIN information_schema.check_constraints cc
  ON cc.constraint_schema = tc.constraint_schema
 AND cc.constraint_name = tc.constraint_name
WHERE tc.table_schema = ?
  AND tc.table_name = ?
ORDER BY tc.constraint_name, k.ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConstraintRows(rows)
}

func scanConstraintRows(rows *sql.Rows) ([]Constraint, error) {
	items := make([]Constraint, 0)
	positions := make(map[string]int)
	for rows.Next() {
		var name, kind, column, definition string
		if err := rows.Scan(&name, &kind, &column, &definition); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s\x00%s", name, kind)
		position, ok := positions[key]
		if !ok {
			position = len(items)
			positions[key] = position
			items = append(items, Constraint{Name: name, Type: kind, Definition: definition, Columns: []string{}})
		}
		if column != "" && !containsMetadataColumn(items[position].Columns, column) {
			items[position].Columns = append(items[position].Columns, column)
		}
		if items[position].Definition == "" && definition != "" {
			items[position].Definition = definition
		}
	}
	return items, rows.Err()
}

func containsMetadataColumn(columns []string, target string) bool {
	for _, column := range columns {
		if column == target {
			return true
		}
	}
	return false
}
