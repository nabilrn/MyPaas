package port

import (
	"context"
	"net"
	"strconv"
	"testing"
)

func TestCanBindUsesDockerPortBindingsForNonLocalBindHost(t *testing.T) {
	t.Setenv("DOCKER_BIND_HOST", "172.18.0.1")
	original := dockerPortBindings
	dockerPortBindings = func(context.Context) ([]dockerInspectPortRow, error) {
		return []dockerInspectPortRow{
			{
				NetworkSettings: struct {
					Ports map[string][]struct {
						HostIP   string `json:"HostIp"`
						HostPort string `json:"HostPort"`
					} `json:"Ports"`
				}{
					Ports: map[string][]struct {
						HostIP   string `json:"HostIp"`
						HostPort string `json:"HostPort"`
					}{
						"3001/tcp": {{HostIP: "172.18.0.1", HostPort: "3001"}},
					},
				},
			},
		}, nil
	}
	t.Cleanup(func() { dockerPortBindings = original })

	if canBind(3001) {
		t.Fatal("canBind(3001) = true for docker-bound host port")
	}
	if !canBind(3004) {
		t.Fatal("canBind(3004) = false for available bridge host port")
	}
}

func TestCanBindUsesDockerBindHost(t *testing.T) {
	t.Setenv("DOCKER_BIND_HOST", "127.0.0.1")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen test port: %v", err)
	}
	defer ln.Close()

	port := int32(ln.Addr().(*net.TCPAddr).Port)
	if canBind(port) {
		t.Fatalf("canBind(%d) = true while %s is already bound", port, net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port))))
	}
}
