package migration

import (
	"reflect"
	"testing"
)

func TestProjectEngineVolumesReturnsOnlyMyPaasComposeVolumes(t *testing.T) {
	raw := []byte(`[
  {
    "Config": {"Labels": {"com.docker.compose.project": "mypaas-blog"}},
    "Mounts": [
      {"Type": "volume", "Name": "mypaas-blog_db"},
      {"Type": "bind", "Name": ""}
    ]
  },
  {
    "Config": {"Labels": {"com.docker.compose.project": "mypaas-blog"}},
    "Mounts": [{"Type": "volume", "Name": "mypaas-blog_db"}]
  },
  {
    "Config": {"Labels": {"com.docker.compose.project": "mypaas-shop"}},
    "Mounts": [{"Type": "volume", "Name": "shared-data"}]
  },
  {
    "Config": {"Labels": {"com.docker.compose.project": "mypaas"}},
    "Mounts": [{"Type": "volume", "Name": "mypaas_postgres_data"}]
  },
  {
    "Config": {"Labels": {}},
    "Mounts": [{"Type": "volume", "Name": "unrelated"}]
  }
]`)

	got, err := projectEngineVolumes(raw)
	if err != nil {
		t.Fatalf("projectEngineVolumes() error = %v", err)
	}
	want := []string{"mypaas-blog_db", "shared-data"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("projectEngineVolumes() = %#v, want %#v", got, want)
	}
}

func TestProjectEngineVolumesRejectsInvalidInspectJSON(t *testing.T) {
	if _, err := projectEngineVolumes([]byte(`not-json`)); err == nil {
		t.Fatal("projectEngineVolumes() must reject invalid docker inspect JSON")
	}
}
