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
    "Config": {"Labels": {"com.docker.compose.project": "mypaas-pr32", "com.docker.compose.service": "postgres"}},
    "Mounts": [{"Type": "volume", "Name": "mypaas-pr32_postgres_data"}]
  },
  {
    "Config": {"Labels": {"com.docker.compose.project": "mypaas-pr32", "com.docker.compose.service": "caddy"}},
    "Mounts": [{"Type": "volume", "Name": "mypaas-pr32_caddy_data"}]
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

func TestProjectEngineVolumeMountsKeepsCopySourceAndDetectsAnyRunningConsumer(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "compose-db",
    "State": {"Running": false},
    "Config": {"Labels": {"com.docker.compose.project": "mypaas-wago", "com.docker.compose.service": "db"}},
    "Mounts": [{"Type": "volume", "Name": "162ef25e-2566-49b2-b7a7-92ddb1b9af62_wago_data", "Destination": "/var/lib/postgresql/data"}]
  },
  {
    "Id": "unexpected-consumer",
    "State": {"Running": true},
    "Config": {"Labels": {}},
    "Mounts": [{"Type": "volume", "Name": "162ef25e-2566-49b2-b7a7-92ddb1b9af62_wago_data", "Destination": "/data"}]
  },
  {
    "Id": "platform-postgres",
    "State": {"Running": true},
    "Config": {"Labels": {"com.docker.compose.project": "mypaas-prod", "com.docker.compose.service": "postgres"}},
    "Mounts": [{"Type": "volume", "Name": "platform_data", "Destination": "/var/lib/postgresql/data"}]
  }
]`)

	got, err := projectEngineVolumeMounts(raw)
	if err != nil {
		t.Fatalf("projectEngineVolumeMounts() error = %v", err)
	}
	want := []engineVolumeMount{{
		Name:        "162ef25e-2566-49b2-b7a7-92ddb1b9af62_wago_data",
		ContainerID: "compose-db",
		Destination: "/var/lib/postgresql/data",
		Running:     true,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("projectEngineVolumeMounts() = %#v, want %#v", got, want)
	}
}

func TestProjectEngineVolumeMountsRejectsUnsafeVolumeName(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "compose-db",
    "State": {"Running": false},
    "Config": {"Labels": {"com.docker.compose.project": "mypaas-demo"}},
    "Mounts": [{"Type": "volume", "Name": "../escape", "Destination": "/data"}]
  }
]`)
	if _, err := projectEngineVolumeMounts(raw); err == nil {
		t.Fatal("projectEngineVolumeMounts() must reject unsafe volume names")
	}
}
