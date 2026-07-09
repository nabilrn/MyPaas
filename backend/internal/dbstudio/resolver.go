package dbstudio

import (
	"context"
	"database/sql"
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
		if err := connectComposeNetwork(ctx, project.Name); err != nil {
			return Connection{}, err
		}
	}
	return conn, nil
}

func connectionWithDSN(conn Connection, password string, query url.Values) Connection {
	switch conn.Driver {
	case DriverPostgres:
		conn.DSN = postgresDSN(conn, password, query)
	case DriverMySQL, DriverMariaDB:
		conn.DSN = mysqlDSN(conn, password)
	}
	return conn
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

func connectComposeNetwork(ctx context.Context, projectName string) error {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return nil
	}
	network := "mypaas-" + projectName + "_default"
	out, err := exec.CommandContext(ctx, "docker", "network", "connect", network, hostname).CombinedOutput()
	if err == nil || isAlreadyConnected(string(out)) || isDockerUnavailable(string(out)) {
		return nil
	}
	return fmt.Errorf("%w: connect API to compose network: %s", errs.ErrValidation, firstLine(string(out)))
}

func isAlreadyConnected(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "already exists") || strings.Contains(output, "already connected")
}

func isDockerUnavailable(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "no such container") || strings.Contains(output, "cannot connect to the docker daemon")
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
