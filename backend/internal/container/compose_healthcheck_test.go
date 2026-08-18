package container

import (
	"encoding/json"
	"testing"
)

func TestSanitizeComposeConfigConvertsCMDHealthcheckWithShellSafeQuoting(t *testing.T) {
	raw := []byte(`{
		"services": {
			"web": {
				"image": "demo",
				"healthcheck": {
					"test": ["CMD", "curl", "-fsS", "http://localhost:8080/health?token=$abc&x=1", "hello world", "quote'it", ""]
				}
			}
		}
	}`)

	out, err := sanitizeComposeConfig(raw)
	if err != nil {
		t.Fatalf("sanitizeComposeConfig() error = %v", err)
	}

	var doc struct {
		Services map[string]struct {
			Healthcheck struct {
				Test []any `json:"test"`
			} `json:"healthcheck"`
		} `json:"services"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("sanitized json invalid: %v", err)
	}

	got := doc.Services["web"].Healthcheck.Test
	if len(got) != 2 {
		t.Fatalf("healthcheck test = %#v, want two entries", got)
	}
	if got[0] != "CMD-SHELL" {
		t.Fatalf("healthcheck mode = %#v, want CMD-SHELL", got[0])
	}
	want := `'curl' '-fsS' 'http://localhost:8080/health?token=$abc&x=1' 'hello world' 'quote'\''it' ''`
	if got[1] != want {
		t.Fatalf("healthcheck command = %#v, want %#v", got[1], want)
	}
}

func TestSanitizeComposeConfigPreservesExistingCMDShellHealthcheck(t *testing.T) {
	raw := []byte(`{
		"services": {
			"web": {
				"image": "demo",
				"healthcheck": {
					"test": ["CMD-SHELL", "curl -fsS $URL || exit 1"]
				}
			}
		}
	}`)

	out, err := sanitizeComposeConfig(raw)
	if err != nil {
		t.Fatalf("sanitizeComposeConfig() error = %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("sanitized json invalid: %v", err)
	}
	test := doc["services"].(map[string]any)["web"].(map[string]any)["healthcheck"].(map[string]any)["test"].([]any)
	if test[0] != "CMD-SHELL" || test[1] != "curl -fsS $URL || exit 1" {
		t.Fatalf("existing CMD-SHELL healthcheck changed: %#v", test)
	}
}
