package dbstudio

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestSQLiteRuntimeDiscoveryUsesRuntimeIdentity(t *testing.T) {
	previousRuntimeCommand := sqliteRuntimeCommand
	previousHelperCommand := sqliteRuntimeHelperCommand

	sqliteRuntimeCommand = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) == 4 && args[0] == "exec" && args[1] == "mypaas-wago-wago-1" && args[2] == "id" {
			switch args[3] {
			case "-u", "-g":
				return []byte("1000\n"), nil
			}
		}
		return nil, fmt.Errorf("unexpected runtime command: %v", args)
	}

	sqliteRuntimeHelperCommand = func(_ context.Context, args []string, payload []byte) ([]byte, error) {
		var runtimeUser string
		var volumesFrom string
		for index := 0; index+1 < len(args); index++ {
			switch args[index] {
			case "--user":
				runtimeUser = args[index+1]
			case "--volumes-from":
				volumesFrom = args[index+1]
			}
		}
		if runtimeUser != "1000:1000" {
			return nil, fmt.Errorf("discovery helper user = %q; want 1000:1000", runtimeUser)
		}
		if volumesFrom != "mypaas-wago-wago-1" {
			return nil, fmt.Errorf("discovery helper volumes-from = %q", volumesFrom)
		}

		var request SQLiteHelperRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if request.Operation != "discover" || len(request.DiscoveryRoots) != 1 || request.DiscoveryRoots[0] != "/app/data" {
			return nil, fmt.Errorf("unexpected discovery request: %#v", request)
		}
		return []byte(`{"ok":true,"response":{"sqliteCandidates":["/app/data/wago.db"]}}`), nil
	}

	t.Setenv("MYPAAS_SQLITE_HELPER_IMAGE", "mypaas-api:test")
	t.Cleanup(func() {
		sqliteRuntimeCommand = previousRuntimeCommand
		sqliteRuntimeHelperCommand = previousHelperCommand
	})

	client := &sqliteRuntimeClient{container: "mypaas-wago-wago-1"}
	paths, err := client.Discover(context.Background(), []string{"/app/data"})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "/app/data/wago.db" {
		t.Fatalf("discovery paths = %#v; want [/app/data/wago.db]", paths)
	}
}
