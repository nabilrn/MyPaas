package dbstudio

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	mysqlcfg "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"mypaas/internal/db"
	"mypaas/internal/errs"
)

func resolveConnection(ctx context.Context, project db.Project, envs map[string]string) (Connection, error) {
	if conn, ok := resolveURL(envs); ok {
		return prepareComposeConnection(ctx, project, conn)
	}
	if conn, ok := resolveParts(project, envs); ok {
		return prepareComposeConnection(ctx, project, conn)
	}
	return Connection{}, fmt.Errorf("%w: no supported database environment was found", errs.ErrValidation)
}

func resolveURL(envs map[string]string) (Connection, bool) {
	for _, key := range []string{"DATABASE_URL", "POSTGRES_URL", "POSTGRESQL_URL", "MYSQL_URL", "MARIADB_URL"} {
		raw := strings.TrimSpace(envs[key])
		if raw == "" {
			continue
		}
		conn, err := connectionFromURL(raw, key)
		if err == nil {
			return conn, true
		}
	}
	return Connection{}, false
}

func connectionFromURL(raw, source string) (Connection, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return Connection{}, err
	}
	driver, ok := driverFromScheme(parsed.Scheme)
	if !ok {
		return Connection{}, fmt.Errorf("unsupported database scheme %q", parsed.Scheme)
	}
	port := defaultPort(driver)
	if parsed.Port() != "" {
		if value, err := strconv.Atoi(parsed.Port()); err == nil {
			port = value
		}
	}
	password, _ := parsed.User.Password()
	return connectionWithDSN(Connection{
		Driver: driver, Host: parsed.Hostname(), Port: port,
		Database: strings.TrimPrefix(parsed.Path, "/"), User: parsed.User.Username(), Source: source,
	}, password, parsed.Query()), nil
}

func resolveParts(project db.Project, envs map[string]string) (Connection, bool) {
	driver := inferDriver(envs)
	if driver == "" {
		return Connection{}, false
	}
	host := firstEnv(envs, "DB_HOST", "DATABASE_HOST", "POSTGRES_HOST", "MYSQL_HOST", "MARIADB_HOST")
	if host == "" && project.DeployMode == "compose" {
		host = "db"
	}
	port := intFromEnv(envs, defaultPort(driver), "DB_PORT", "DATABASE_PORT", "POSTGRES_PORT", "MYSQL_PORT", "MARIADB_PORT")
	name := firstEnv(envs, "DB_NAME", "DATABASE_NAME", "POSTGRES_DB", "MYSQL_DATABASE", "MARIADB_DATABASE")
	user := firstEnv(envs, "DB_USER", "DATABASE_USER", "POSTGRES_USER", "MYSQL_USER", "MARIADB_USER")
	pass := firstEnv(envs, "DB_PASSWORD", "DATABASE_PASSWORD", "POSTGRES_PASSWORD", "MYSQL_PASSWORD", "MARIADB_PASSWORD")
	if host == "" || name == "" || user == "" {
		return Connection{}, false
	}
	conn := Connection{Driver: driver, Host: host, Port: port, Database: name, User: user, Source: "env-parts"}
	return connectionWithDSN(conn, pass, nil), true
}

func inferDriver(envs map[string]string) DriverID {
	hint := strings.ToLower(firstEnv(envs, "DB_CONNECTION", "DB_DRIVER", "DATABASE_CLIENT"))
	switch {
	case strings.Contains(hint, "postgres"), envs["POSTGRES_DB"] != "" || envs["POSTGRES_USER"] != "":
		return DriverPostgres
	case strings.Contains(hint, "mariadb"), envs["MARIADB_DATABASE"] != "":
		return DriverMariaDB
	case strings.Contains(hint, "mysql"), envs["MYSQL_DATABASE"] != "":
		return DriverMySQL
	case envs["DB_ROOT_PASSWORD"] != "":
		return DriverMariaDB
	case envs["DB_PORT"] == "5432":
		return DriverPostgres
	case envs["DB_PORT"] == "3306":
		return DriverMySQL
	default:
		return ""
	}
}

func prepareComposeConnection(ctx context.Context, project db.Project, conn Connection) (Connection, error) {
	if project.DeployMode == "compose" && shouldConnectComposeNetwork(conn.Host) {
		if retargeted, ok, err := retargetComposeService(ctx, project.Name, conn); err != nil {
			return Connection{}, err
		} else if ok {
			return retargeted, nil
		}
		if err := connectComposeNetworks(ctx, project.Name); err != nil {
			return Connection{}, err
		}
	}
	return conn, nil
}

func connectionWithDSN(conn Connection, password string, query url.Values) Connection {
	conn.password = password
	conn.query = cloneQuery(query)
	switch conn.Driver {
	case DriverPostgres:
		conn.DSN = postgresDSN(conn, password, query)
	case DriverMySQL, DriverMariaDB:
		conn.DSN = mysqlDSN(conn, password)
	}
	return conn
}

func cloneQuery(query url.Values) map[string][]string {
	if len(query) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(query))
	for key, values := range query {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func retargetConnection(conn Connection, host string) Connection {
	conn.Host = host
	if conn.Source != "" && !strings.Contains(conn.Source, "+compose-ip") {
		conn.Source += "+compose-ip"
	}
	return connectionWithDSN(conn, conn.password, url.Values(conn.query))
}

func postgresDSN(conn Connection, password string, query url.Values) string {
	u := url.URL{Scheme: "postgres", User: url.UserPassword(conn.User, password), Host: net.JoinHostPort(conn.Host, fmt.Sprint(conn.Port)), Path: "/" + conn.Database}
	q := u.Query()
	for key, values := range query {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	if q.Get("sslmode") == "" {
		q.Set("sslmode", "disable")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func mysqlDSN(conn Connection, password string) string {
	config := mysqlcfg.NewConfig()
	config.User = conn.User
	config.Passwd = password
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(conn.Host, fmt.Sprint(conn.Port))
	config.DBName = conn.Database
	config.ParseTime = true
	config.Timeout = 5 * time.Second
	config.ReadTimeout = 5 * time.Second
	config.WriteTimeout = 5 * time.Second
	return config.FormatDSN()
}

func openConnection(ctx context.Context, conn Connection) (*sql.DB, Adapter, error) {
	driver, adapter := sqlDriver(conn.Driver)
	handle, err := sql.Open(driver, conn.DSN)
	if err != nil {
		return nil, nil, err
	}
	handle.SetMaxOpenConns(3)
	handle.SetMaxIdleConns(1)
	handle.SetConnMaxLifetime(5 * time.Minute)
	if err := handle.PingContext(ctx); err != nil {
		handle.Close()
		return nil, nil, err
	}
	return handle, adapter, nil
}

func sqlDriver(driver DriverID) (string, Adapter) {
	switch driver {
	case DriverPostgres:
		return "pgx", postgresAdapter{}
	default:
		return "mysql", mysqlAdapter{}
	}
}

func retargetComposeService(ctx context.Context, projectName string, conn Connection) (Connection, bool, error) {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return conn, false, nil
	}

	target, ok, err := composeServiceTarget(ctx, projectName, conn.Host)
	if err != nil {
		return Connection{}, false, err
	}
	if !ok {
		return conn, false, nil
	}
	connected, err := connectDockerNetwork(ctx, target.Network, hostname)
	if err != nil {
		return Connection{}, false, err
	}
	if !connected {
		return conn, false, nil
	}
	return retargetConnection(conn, target.IPAddress), true, nil
}

type composeServiceEndpoint struct {
	Network   string
	IPAddress string
}

func composeServiceTarget(ctx context.Context, projectName, service string) (composeServiceEndpoint, bool, error) {
	service = strings.TrimSpace(service)
	if service == "" {
		return composeServiceEndpoint{}, false, nil
	}
	for _, project := range composeProjectCandidates(projectName) {
		ids, err := composeServiceContainerIDs(ctx, project, service)
		if err != nil {
			return composeServiceEndpoint{}, false, err
		}
		for _, id := range ids {
			endpoints, err := inspectContainerNetworks(ctx, id)
			if err != nil {
				return composeServiceEndpoint{}, false, err
			}
			if endpoint, ok := preferredEndpoint(endpoints); ok {
				return endpoint, true, nil
			}
		}
	}
	return composeServiceEndpoint{}, false, nil
}

func composeServiceContainerIDs(ctx context.Context, projectName, service string) ([]string, error) {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-aq",
		"--filter", "label=com.docker.compose.project="+projectName,
		"--filter", "label=com.docker.compose.service="+service,
	).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if isDockerUnavailable(msg) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: find compose service container: %s", errs.ErrValidation, firstLine(msg))
	}
	return fieldsByLine(string(out)), nil
}

func inspectContainerNetworks(ctx context.Context, containerID string) ([]composeServiceEndpoint, error) {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{json .NetworkSettings.Networks}}", containerID).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if isDockerUnavailable(msg) || isNoSuchContainer(msg) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: inspect compose service networks: %s", errs.ErrValidation, firstLine(msg))
	}
	return parseComposeNetworkInspect(string(out))
}

func parseComposeNetworkInspect(value string) ([]composeServiceEndpoint, error) {
	var raw map[string]struct {
		IPAddress string `json:"IPAddress"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &raw); err != nil {
		return nil, fmt.Errorf("%w: parse compose service networks: %v", errs.ErrValidation, err)
	}
	endpoints := make([]composeServiceEndpoint, 0, len(raw))
	for network, item := range raw {
		network = strings.TrimSpace(network)
		ip := strings.TrimSpace(item.IPAddress)
		if network == "" || ip == "" {
			continue
		}
		endpoints = append(endpoints, composeServiceEndpoint{Network: network, IPAddress: ip})
	}
	return endpoints, nil
}

func preferredEndpoint(endpoints []composeServiceEndpoint) (composeServiceEndpoint, bool) {
	for _, endpoint := range endpoints {
		if !strings.Contains(strings.ToLower(endpoint.Network), "mypaas_platform") {
			return endpoint, true
		}
	}
	if len(endpoints) == 0 {
		return composeServiceEndpoint{}, false
	}
	return endpoints[0], true
}

func connectComposeNetworks(ctx context.Context, projectName string) error {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return nil
	}
	networks, err := composeProjectNetworks(ctx, projectName)
	if err != nil {
		return err
	}
	if len(networks) == 0 {
		networks = fallbackComposeNetworks(projectName)
	}
	for _, network := range networks {
		if _, err := connectDockerNetwork(ctx, network, hostname); err != nil {
			return err
		}
	}
	return nil
}

func composeProjectNetworks(ctx context.Context, projectName string) ([]string, error) {
	seen := make(map[string]struct{})
	networks := make([]string, 0)
	for _, project := range composeProjectCandidates(projectName) {
		out, err := exec.CommandContext(ctx, "docker", "network", "ls", "-q", "--filter", "label=com.docker.compose.project="+project).CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if isDockerUnavailable(msg) {
				return nil, nil
			}
			return nil, fmt.Errorf("%w: list compose networks: %s", errs.ErrValidation, firstLine(msg))
		}
		for _, network := range fieldsByLine(string(out)) {
			if _, ok := seen[network]; ok {
				continue
			}
			seen[network] = struct{}{}
			networks = append(networks, network)
		}
	}
	return networks, nil
}

func connectDockerNetwork(ctx context.Context, network, container string) (bool, error) {
	network = strings.TrimSpace(network)
	container = strings.TrimSpace(container)
	if network == "" || container == "" {
		return false, nil
	}
	out, err := exec.CommandContext(ctx, "docker", "network", "connect", network, container).CombinedOutput()
	msg := string(out)
	if err == nil || isAlreadyConnected(msg) {
		return true, nil
	}
	if isDockerUnavailable(msg) || isNoSuchContainer(msg) {
		return false, nil
	}
	return false, fmt.Errorf("%w: connect API to compose network %q: %s", errs.ErrValidation, network, firstLine(msg))
}

func composeProjectCandidates(projectName string) []string {
	projectName = strings.TrimSpace(projectName)
	if projectName == "" {
		return nil
	}
	candidates := []string{projectName}
	if !strings.HasPrefix(projectName, "mypaas-") {
		candidates = append([]string{"mypaas-" + projectName}, candidates...)
	}
	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

func fallbackComposeNetworks(projectName string) []string {
	out := make([]string, 0, len(composeProjectCandidates(projectName)))
	for _, project := range composeProjectCandidates(projectName) {
		out = append(out, project+"_default")
	}
	return out
}

func isAlreadyConnected(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "already exists") || strings.Contains(output, "already connected")
}

func isDockerUnavailable(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "no such container") || strings.Contains(output, "cannot connect to the docker daemon")
}

func isNoSuchContainer(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "no such container")
}

func shouldConnectComposeNetwork(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host != "" && host != "localhost" && host != "127.0.0.1" && host != "host.docker.internal"
}

func driverFromScheme(scheme string) (DriverID, bool) {
	switch strings.ToLower(scheme) {
	case "postgres", "postgresql":
		return DriverPostgres, true
	case "mysql":
		return DriverMySQL, true
	case "mariadb":
		return DriverMariaDB, true
	default:
		return "", false
	}
}

func defaultPort(driver DriverID) int {
	if driver == DriverPostgres {
		return 5432
	}
	return 3306
}

func firstEnv(envs map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(envs[key]); value != "" {
			return value
		}
	}
	return ""
}

func intFromEnv(envs map[string]string, fallback int, keys ...string) int {
	raw := firstEnv(envs, keys...)
	if value, err := strconv.Atoi(raw); err == nil && value > 0 {
		return value
	}
	return fallback
}

func firstLine(value string) string {
	for _, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return "unknown error"
}

func fieldsByLine(value string) []string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
