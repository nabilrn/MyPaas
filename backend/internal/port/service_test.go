package port

import (
	"net"
	"strconv"
	"testing"
)

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
