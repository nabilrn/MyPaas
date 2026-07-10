package dbstudio

import (
	"strings"
	"testing"
)

func TestParseMySQLEnumValues(t *testing.T) {
	got := parseMySQLEnumValues(`enum('draft','in review','it\'s done')`)
	want := []string{"draft", "in review", "it's done"}
	if len(got) != len(want) {
		t.Fatalf("values length = %d, want %d: %#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("value[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestMySQLSelectSQLAppliesSearchAndEnumFilter(t *testing.T) {
	columns := []Column{
		{Name: "id", DataType: "int", PrimaryKey: true},
		{Name: "title", DataType: "varchar(255)"},
		{Name: "status", DataType: "enum('draft','published')", EnumValues: []string{"draft", "published"}},
	}

	sqlText, args, err := mysqlSelectSQL("app", "posts", columns, RowQuery{
		Search:  "hello_% world",
		Filters: map[string]string{"status": "draft"},
	}, 100, 0)
	if err != nil {
		t.Fatalf("mysqlSelectSQL() error = %v", err)
	}
	if !strings.Contains(sqlText, "WHERE `status` = ? AND (") {
		t.Fatalf("expected enum filter WHERE clause, got %s", sqlText)
	}
	if !strings.Contains(sqlText, "CAST(`title` AS CHAR) LIKE ?") {
		t.Fatalf("expected SQL-level search, got %s", sqlText)
	}
	if len(args) != 6 {
		t.Fatalf("args length = %d, want 6: %#v", len(args), args)
	}
	if args[0] != "draft" || args[1] != `%hello\_\% world%` {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestPostgresSelectSQLRejectsInvalidEnumFilter(t *testing.T) {
	_, _, err := postgresSelectSQL("public", "posts", []Column{
		{Name: "status", DataType: "USER-DEFINED", EnumValues: []string{"draft", "published"}},
	}, RowQuery{Filters: map[string]string{"status": "deleted"}}, 100, 0)
	if err == nil {
		t.Fatal("postgresSelectSQL() expected invalid enum filter error")
	}
}
