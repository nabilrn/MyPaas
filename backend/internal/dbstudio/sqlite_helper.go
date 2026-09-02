package dbstudio

import (
	"context"
	"fmt"
	pathpkg "path"
	"strings"
)

type SQLiteHelperRequest struct {
	DatabasePath string    `json:"databasePath"`
	Operation    string    `json:"operation"`
	Schema       string    `json:"schema,omitempty"`
	Table        string    `json:"table,omitempty"`
	Query        *RowQuery `json:"query,omitempty"`
	Mutation     *Mutation `json:"mutation,omitempty"`
}

type SQLiteHelperResponse struct {
	Schemas      []Schema      `json:"schemas,omitempty"`
	Tables       []Table       `json:"tables,omitempty"`
	Columns      []Column      `json:"columns,omitempty"`
	TableDetails *TableDetails `json:"tableDetails,omitempty"`
	Rows         *RowPage      `json:"rows,omitempty"`
}

func ExecuteSQLiteHelper(ctx context.Context, request SQLiteHelperRequest) (SQLiteHelperResponse, error) {
	databasePath := strings.TrimSpace(request.DatabasePath)
	if databasePath == "" || !pathpkg.IsAbs(databasePath) || strings.ContainsRune(databasePath, '\x00') {
		return SQLiteHelperResponse{}, fmt.Errorf("invalid SQLite database path")
	}
	conn, err := openSQLiteDatabase(ctx, databasePath)
	if err != nil {
		return SQLiteHelperResponse{}, err
	}
	defer conn.Close()

	adapter := sqliteAdapter{}
	switch request.Operation {
	case "ping":
		var value int
		if err := conn.QueryRowContext(ctx, "SELECT 1").Scan(&value); err != nil {
			return SQLiteHelperResponse{}, err
		}
		return SQLiteHelperResponse{}, nil
	case "schemas":
		items, err := adapter.Schemas(ctx, conn)
		return SQLiteHelperResponse{Schemas: items}, err
	case "tables":
		items, err := adapter.Tables(ctx, conn, request.Schema)
		return SQLiteHelperResponse{Tables: items}, err
	case "columns":
		items, err := adapter.Columns(ctx, conn, request.Schema, request.Table)
		return SQLiteHelperResponse{Columns: items}, err
	case "table_details":
		item, err := adapter.TableDetails(ctx, conn, request.Schema, request.Table)
		if err != nil {
			return SQLiteHelperResponse{}, err
		}
		return SQLiteHelperResponse{TableDetails: &item}, nil
	case "rows":
		if request.Query == nil {
			return SQLiteHelperResponse{}, fmt.Errorf("row query is required")
		}
		item, err := adapter.Rows(ctx, conn, *request.Query)
		if err != nil {
			return SQLiteHelperResponse{}, err
		}
		return SQLiteHelperResponse{Rows: &item}, nil
	case "insert":
		if request.Mutation == nil {
			return SQLiteHelperResponse{}, fmt.Errorf("mutation is required")
		}
		return SQLiteHelperResponse{}, adapter.Insert(ctx, conn, *request.Mutation)
	case "update":
		if request.Mutation == nil {
			return SQLiteHelperResponse{}, fmt.Errorf("mutation is required")
		}
		return SQLiteHelperResponse{}, adapter.Update(ctx, conn, *request.Mutation)
	case "delete":
		if request.Mutation == nil {
			return SQLiteHelperResponse{}, fmt.Errorf("mutation is required")
		}
		return SQLiteHelperResponse{}, adapter.Delete(ctx, conn, *request.Mutation)
	default:
		return SQLiteHelperResponse{}, fmt.Errorf("unsupported SQLite helper operation %q", request.Operation)
	}
}
