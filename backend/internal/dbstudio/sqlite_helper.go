package dbstudio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxSQLiteDiscoveryRoots            = 16
	maxSQLiteDiscoveryDepth            = 3
	maxSQLiteDiscoveryEntries          = 4096
	maxSQLiteDiscoveryCandidates       = 16
	maxSQLiteCandidateBytes      int64 = 512 * 1024 * 1024
)

type SQLiteHelperRequest struct {
	DatabasePath   string    `json:"databasePath"`
	Operation      string    `json:"operation"`
	DiscoveryRoots []string  `json:"discoveryRoots,omitempty"`
	Schema         string    `json:"schema,omitempty"`
	Table          string    `json:"table,omitempty"`
	Query          *RowQuery `json:"query,omitempty"`
	Mutation       *Mutation `json:"mutation,omitempty"`
}

type SQLiteHelperResponse struct {
	Schemas          []Schema      `json:"schemas,omitempty"`
	Tables           []Table       `json:"tables,omitempty"`
	Columns          []Column      `json:"columns,omitempty"`
	TableDetails     *TableDetails `json:"tableDetails,omitempty"`
	Rows             *RowPage      `json:"rows,omitempty"`
	SQLiteCandidates []string      `json:"sqliteCandidates,omitempty"`
}

func ExecuteSQLiteHelper(ctx context.Context, request SQLiteHelperRequest) (SQLiteHelperResponse, error) {
	if request.Operation == "discover" {
		candidates, err := discoverSQLiteFiles(ctx, request.DiscoveryRoots)
		return SQLiteHelperResponse{SQLiteCandidates: candidates}, err
	}
	databasePath := strings.TrimSpace(request.DatabasePath)
	if databasePath == "" || !pathpkg.IsAbs(databasePath) || strings.ContainsRune(databasePath, '\x00') {
		return SQLiteHelperResponse{}, fmt.Errorf("invalid SQLite database path")
	}
	conn, err := openSQLiteDatabase(ctx, databasePath)
	if err != nil {
		return SQLiteHelperResponse{}, err
	}
	defer conn.Close()
	// Table metadata can require nested PRAGMA reads while the index-list rows are
	// still open. Allow a small bounded pool; SQLite locking remains enforced by
	// the database itself and busy_timeout.
	conn.SetMaxOpenConns(4)

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

func discoverSQLiteFiles(ctx context.Context, roots []string) ([]string, error) {
	if len(roots) == 0 {
		return nil, nil
	}
	if len(roots) > maxSQLiteDiscoveryRoots {
		return nil, fmt.Errorf("too many SQLite discovery roots")
	}

	normalizedRoots := make([]string, 0, len(roots))
	seenRoots := make(map[string]struct{}, len(roots))
	for _, rawRoot := range roots {
		root, err := normalizeSQLiteDiscoveryRoot(rawRoot)
		if err != nil {
			return nil, err
		}
		if _, ok := seenRoots[root]; ok {
			continue
		}
		seenRoots[root] = struct{}{}
		normalizedRoots = append(normalizedRoots, root)
	}
	sort.Strings(normalizedRoots)

	candidates := make([]string, 0)
	for _, root := range normalizedRoots {
		rootCandidates, err := discoverSQLiteFilesInRoot(ctx, root)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, rootCandidates...)
		if len(candidates) > maxSQLiteDiscoveryCandidates {
			return nil, fmt.Errorf("too many SQLite database candidates")
		}
	}
	sort.Strings(candidates)
	return candidates, nil
}

func discoverSQLiteFilesInRoot(ctx context.Context, root string) ([]string, error) {
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect SQLite discovery root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("SQLite discovery root must be a directory")
	}

	candidates := make([]string, 0)
	entries := 0
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return fmt.Errorf("walk SQLite discovery root %q: %w", root, walkErr)
		}
		entries++
		if entries > maxSQLiteDiscoveryEntries {
			return fmt.Errorf("SQLite discovery entry limit exceeded")
		}

		depth, err := sqliteDiscoveryDepth(root, path)
		if err != nil {
			return err
		}
		if depth > maxSQLiteDiscoveryDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !isSQLiteCandidateName(entry.Name()) {
			return nil
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect SQLite discovery entry %q: %w", path, err)
		}
		if !fileInfo.Mode().IsRegular() || fileInfo.Size() < 100 || fileInfo.Size() > maxSQLiteCandidateBytes {
			return nil
		}
		valid, err := hasSQLiteHeader(path)
		if err != nil {
			return fmt.Errorf("read SQLite discovery candidate %q: %w", path, err)
		}
		if valid {
			candidates = append(candidates, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return candidates, nil
}

func normalizeSQLiteDiscoveryRoot(raw string) (string, error) {
	root := strings.TrimSpace(raw)
	if root == "" || (!pathpkg.IsAbs(root) && !filepath.IsAbs(root)) {
		return "", fmt.Errorf("SQLite discovery root must be absolute")
	}
	root = filepath.Clean(root)
	if root == "." || pathpkg.Clean(filepath.ToSlash(root)) == "/" {
		return "", fmt.Errorf("SQLite discovery root cannot be the filesystem root")
	}
	return root, nil
}

func sqliteDiscoveryDepth(root, path string) (int, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return 0, fmt.Errorf("resolve SQLite discovery path: %w", err)
	}
	if relative == "." {
		return 0, nil
	}
	return len(strings.Split(relative, string(filepath.Separator))), nil
}

func isSQLiteCandidateName(name string) bool {
	lowered := strings.ToLower(strings.TrimSpace(name))
	return strings.HasSuffix(lowered, ".db") || strings.HasSuffix(lowered, ".sqlite") || strings.HasSuffix(lowered, ".sqlite3")
}

func hasSQLiteHeader(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	var header [16]byte
	if _, err := io.ReadFull(file, header[:]); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return false, nil
		}
		return false, err
	}
	return bytes.Equal(header[:], []byte("SQLite format 3\x00")), nil
}
