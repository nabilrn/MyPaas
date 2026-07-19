package project

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"mypaas/internal/envdiscover"
	"mypaas/internal/envvar"
	"mypaas/internal/errs"
)

var composeEnvExpr = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?:(:?[-?])([^}]*))?\}`)

type composeEnvMatch struct {
	key          string
	operator     string
	defaultValue string
}

type ComposePlan struct {
	RecommendedMainService string               `json:"recommendedMainService"`
	RecommendedAppPort     int32                `json:"recommendedAppPort"`
	RouteTarget            string               `json:"routeTarget"`
	RequiredEnvVars        []string             `json:"requiredEnvVars"`
	Services               []ComposeServicePlan `json:"services"`
	Issues                 []ComposeIssue       `json:"issues"`
}

type ComposeServicePlan struct {
	Name         string            `json:"name"`
	Role         string            `json:"role"`
	BuildContext *string           `json:"buildContext"`
	Dockerfile   *string           `json:"dockerfile"`
	Image        *string           `json:"image"`
	Ports        []ComposePortPlan `json:"ports"`
	Expose       []int32           `json:"expose"`
	DependsOn    []string          `json:"dependsOn"`
}

type ComposePortPlan struct {
	Target    int32   `json:"target"`
	Published *string `json:"published"`
	Protocol  string  `json:"protocol"`
}

type ComposeIssue struct {
	Severity string  `json:"severity"`
	Code     string  `json:"code"`
	Service  *string `json:"service,omitempty"`
	Message  string  `json:"message"`
}

type composeConfigDoc struct {
	Name     string                          `json:"name"`
	Services map[string]composeServiceConfig `json:"services"`
	Networks map[string]struct {
		External bool `json:"external"`
	} `json:"networks"`
}

type composeServiceConfig struct {
	Image       string          `json:"image"`
	Build       json.RawMessage `json:"build"`
	Ports       json.RawMessage `json:"ports"`
	Expose      json.RawMessage `json:"expose"`
	DependsOn   json.RawMessage `json:"depends_on"`
	Environment json.RawMessage `json:"environment"`
	Volumes     json.RawMessage `json:"volumes"`
	Networks    json.RawMessage `json:"networks"`
	NetworkMode string          `json:"network_mode"`
	Privileged  bool            `json:"privileged"`
	Container   string          `json:"container_name"`
}

type composeBuildConfig struct {
	Context    string `json:"context"`
	Dockerfile string `json:"dockerfile"`
}

func inspectComposePlan(ctx context.Context, workspace, composeFile string, services []string, mainService string, appPort int32, envVars []envdiscover.Var) (*ComposePlan, error) {
	rawConfig, err := composeConfigJSON(ctx, workspace, composeFile)
	if err != nil {
		return nil, err
	}
	var doc composeConfigDoc
	if err := json.Unmarshal(rawConfig, &doc); err != nil {
		return nil, fmt.Errorf("parse compose config json: %w", err)
	}
	if len(doc.Services) == 0 {
		return nil, fmt.Errorf("%w: compose file does not define any services", errs.ErrValidation)
	}

	// Docker Compose resolves build.context relative to the compose file's
	// directory, not the cwd. Mirror that here so subdir compose files get
	// accurate build-context existence checks.
	composeDir := filepath.Dir(filepath.Join(workspace, composeFile))

	plan := &ComposePlan{
		RecommendedMainService: mainService,
		RecommendedAppPort:     appPort,
		RouteTarget:            fmt.Sprintf("%s:%d", mainService, appPort),
		RequiredEnvVars:        requiredComposeEnvVars(workspace, composeFile, envVars),
		Services:               make([]ComposeServicePlan, 0, len(services)),
		Issues:                 make([]ComposeIssue, 0),
	}

	for _, serviceName := range services {
		spec, ok := doc.Services[serviceName]
		if !ok {
			continue
		}
		item := composeServicePlanFromConfig(composeDir, serviceName, spec)
		if serviceName == mainService {
			item.Role = "public"
		} else {
			item.Role = "internal"
		}
		plan.Services = append(plan.Services, item)
		addComposeServiceIssues(plan, composeDir, serviceName, item, spec)
	}
	addComposePlanIssues(plan, doc, mainService, appPort)
	sortComposeIssues(plan.Issues)
	return plan, nil
}

func composeConfigJSON(ctx context.Context, workspace, composeFile string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "config", "--format", "json")
	cmd.Dir = workspace
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = firstNonEmptyLine(string(out))
		}
		return nil, fmt.Errorf("%w: compose config could not be validated: %s", errs.ErrValidation, firstNonEmptyLine(message))
	}
	return out, nil
}

func prepareComposePreviewEnv(workspace, composeFile string, envVars []envdiscover.Var) error {
	envPath := filepath.Join(workspace, ".env")
	if pathExists(envPath) {
		return nil
	}
	values := make(map[string]string)
	for _, item := range envVars {
		if strings.TrimSpace(item.Key) == "" {
			continue
		}
		if item.DefaultValue != nil {
			values[item.Key] = *item.DefaultValue
			continue
		}
		values[item.Key] = composePreviewEnvValue(item.Key)
	}
	for _, key := range composeVariableKeys(workspace, composeFile) {
		if _, ok := values[key]; ok {
			continue
		}
		values[key] = composePreviewEnvValue(key)
	}
	return envvar.WriteEnvFile(envPath, values)
}

func composePreviewEnvValue(key string) string {
	upper := strings.ToUpper(strings.TrimSpace(key))
	switch {
	case strings.Contains(upper, "PORT"):
		return "3000"
	case strings.Contains(upper, "URL") && strings.Contains(upper, "DATABASE"):
		return "mysql://mypaas_preview:mypaas_preview@db:3306/mypaas_preview"
	case strings.Contains(upper, "NAME") || strings.Contains(upper, "USER"):
		return "mypaas_preview"
	default:
		return "mypaas_preview"
	}
}

func composeVariableKeys(workspace, composeFile string) []string {
	content, err := os.ReadFile(filepath.Join(workspace, composeFile))
	if err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	for _, match := range composeEnvMatches(string(content)) {
		if strings.TrimSpace(match.key) == "" {
			continue
		}
		seen[match.key] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func composeEnvMatches(content string) []composeEnvMatch {
	indexes := composeEnvExpr.FindAllStringSubmatchIndex(content, -1)
	matches := make([]composeEnvMatch, 0, len(indexes))
	for _, index := range indexes {
		if len(index) < 8 {
			continue
		}
		start := index[0]
		if start > 0 && content[start-1] == '$' {
			continue
		}
		item := composeEnvMatch{key: content[index[2]:index[3]]}
		if index[4] >= 0 && index[5] >= 0 {
			item.operator = content[index[4]:index[5]]
		}
		if index[6] >= 0 && index[7] >= 0 {
			item.defaultValue = content[index[6]:index[7]]
		}
		matches = append(matches, item)
	}
	return matches
}

func composeServicePlanFromConfig(workspace, serviceName string, spec composeServiceConfig) ComposeServicePlan {
	buildContext, dockerfile := composeBuildInfo(spec.Build)
	ports := composePortPlans(spec.Ports)
	expose := composeExposePorts(spec.Expose)
	dependsOn := composeDependsOn(spec.DependsOn)
	image := nullableString(spec.Image)
	item := ComposeServicePlan{
		Name:         serviceName,
		BuildContext: buildContext,
		Dockerfile:   dockerfile,
		Image:        image,
		Ports:        ports,
		Expose:       expose,
		DependsOn:    dependsOn,
	}
	if buildContext != nil {
		_, displayPath := composeBuildContextPath(workspace, *buildContext)
		item.BuildContext = &displayPath
	}
	return item
}

func addComposePlanIssues(plan *ComposePlan, doc composeConfigDoc, mainService string, appPort int32) {
	if mainService == "" {
		plan.Issues = append(plan.Issues, composeIssue("error", "MAIN_SERVICE_REQUIRED", nil, "Main service is required for Compose deploys."))
	}
	if appPort <= 0 {
		plan.Issues = append(plan.Issues, composeIssue("error", "APP_PORT_REQUIRED", &mainService, "App port could not be inferred. Set the container port for the public service."))
	}
	for network, config := range doc.Networks {
		if config.External {
			message := fmt.Sprintf("Network %q is external. Make sure it exists on the MyPaas host or remove external networking.", network)
			plan.Issues = append(plan.Issues, composeIssue("warning", "EXTERNAL_NETWORK", nil, message))
		}
	}
}

func addComposeServiceIssues(plan *ComposePlan, workspace, serviceName string, item ComposeServicePlan, spec composeServiceConfig) {
	if item.BuildContext != nil {
		contextPath, displayPath := composeBuildContextPath(workspace, *item.BuildContext)
		if !pathExists(contextPath) {
			message := fmt.Sprintf("Build context %q does not exist.", displayPath)
			plan.Issues = append(plan.Issues, composeIssue("error", "BUILD_CONTEXT_MISSING", &serviceName, message))
		}
		dockerfile := "Dockerfile"
		if item.Dockerfile != nil && strings.TrimSpace(*item.Dockerfile) != "" {
			dockerfile = *item.Dockerfile
		}
		dockerfilePath := filepath.Join(contextPath, dockerfile)
		if !pathExists(dockerfilePath) {
			message := fmt.Sprintf("Dockerfile %q does not exist in build context %q.", dockerfile, displayPath)
			plan.Issues = append(plan.Issues, composeIssue("error", "DOCKERFILE_MISSING", &serviceName, message))
		}
	}
	if spec.Container != "" {
		plan.Issues = append(plan.Issues, composeIssue("warning", "CONTAINER_NAME", &serviceName, "container_name can collide between deployments; MyPaas compose project names are safer."))
	}
	if spec.NetworkMode == "host" {
		plan.Issues = append(plan.Issues, composeIssue("error", "HOST_NETWORK", &serviceName, "network_mode: host is not compatible with MyPaas routing."))
	}
	if spec.Privileged {
		plan.Issues = append(plan.Issues, composeIssue("error", "PRIVILEGED_CONTAINER", &serviceName, "privileged containers are blocked by MyPaas safety policy."))
	}
	if hasDockerSocketMount(spec.Volumes) {
		plan.Issues = append(plan.Issues, composeIssue("error", "DOCKER_SOCKET_MOUNT", &serviceName, "Mounting /var/run/docker.sock into app containers is not allowed."))
	}
	for _, port := range item.Ports {
		if port.Published == nil || strings.TrimSpace(*port.Published) == "" {
			continue
		}
		if *port.Published == "80" || *port.Published == "443" {
			message := fmt.Sprintf("Publishes host port %s. MyPaas will route through Caddy and replace host port bindings.", *port.Published)
			plan.Issues = append(plan.Issues, composeIssue("warning", "HOST_PORT_RESERVED", &serviceName, message))
			continue
		}
		message := fmt.Sprintf("Publishes host port %s. MyPaas removes Compose host ports and exposes only the selected public service.", *port.Published)
		plan.Issues = append(plan.Issues, composeIssue("info", "HOST_PORT_IGNORED", &serviceName, message))
	}
}

func composeIssue(severity, code string, service *string, message string) ComposeIssue {
	return ComposeIssue{Severity: severity, Code: code, Service: service, Message: message}
}

func composeBuildContextPath(workspace, buildContext string) (string, string) {
	raw := strings.TrimSpace(buildContext)
	if raw == "" {
		raw = "."
	}
	cleaned := filepath.Clean(filepath.FromSlash(raw))
	contextPath := cleaned
	if !filepath.IsAbs(contextPath) {
		contextPath = filepath.Join(workspace, contextPath)
	}
	contextPath = filepath.Clean(contextPath)

	displayPath := filepath.ToSlash(cleaned)
	if rel, err := filepath.Rel(workspace, contextPath); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		displayPath = filepath.ToSlash(rel)
	}
	return contextPath, displayPath
}

func composeBuildInfo(raw json.RawMessage) (*string, *string) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		context := strings.TrimSpace(asString)
		if context == "" {
			return nil, nil
		}
		dockerfile := "Dockerfile"
		return &context, &dockerfile
	}
	var config composeBuildConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, nil
	}
	context := strings.TrimSpace(config.Context)
	if context == "" {
		context = "."
	}
	dockerfile := strings.TrimSpace(config.Dockerfile)
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}
	return &context, &dockerfile
}

func composePortPlans(raw json.RawMessage) []ComposePortPlan {
	if len(raw) == 0 || string(raw) == "null" {
		return []ComposePortPlan{}
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return []ComposePortPlan{}
	}
	ports := make([]ComposePortPlan, 0, len(values))
	for _, value := range values {
		if port := composePortPlan(value); port.Target > 0 || port.Published != nil {
			ports = append(ports, port)
		}
	}
	return ports
}

func composePortPlan(raw json.RawMessage) ComposePortPlan {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err == nil {
		port := ComposePortPlan{Protocol: "tcp"}
		if value, ok := obj["target"]; ok {
			port.Target = int32FromAny(value)
		}
		if value, ok := obj["published"]; ok {
			published := trimPortValue(fmt.Sprint(value))
			if published != "" && published != "<nil>" {
				port.Published = &published
			}
		}
		if value, ok := obj["protocol"]; ok && strings.TrimSpace(fmt.Sprint(value)) != "" {
			port.Protocol = strings.TrimSpace(fmt.Sprint(value))
		}
		return port
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err != nil {
		return ComposePortPlan{}
	}
	parts := strings.Split(asString, ":")
	target := int32FromString(parts[len(parts)-1])
	published := ""
	if len(parts) >= 2 {
		published = trimPortValue(parts[len(parts)-2])
	}
	port := ComposePortPlan{Target: target, Protocol: "tcp"}
	if published != "" {
		port.Published = &published
	}
	return port
}

func composeExposePorts(raw json.RawMessage) []int32 {
	if len(raw) == 0 || string(raw) == "null" {
		return []int32{}
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return []int32{}
	}
	ports := make([]int32, 0, len(values))
	for _, value := range values {
		if port := composeExposePort(value); port > 0 {
			ports = append(ports, port)
		}
	}
	return ports
}

func composeExposePort(raw json.RawMessage) int32 {
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return int32FromString(asString)
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return int32(number)
	}
	return 0
}

func composeDependsOn(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err == nil {
		out := make([]string, 0, len(asMap))
		for key := range asMap {
			out = append(out, key)
		}
		sort.Strings(out)
		return out
	}
	var asList []string
	if err := json.Unmarshal(raw, &asList); err == nil {
		sort.Strings(asList)
		return asList
	}
	return []string{}
}

func hasDockerSocketMount(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	return strings.Contains(string(raw), "/var/run/docker.sock")
}

func requiredComposeEnvVars(workspace, composeFile string, envVars []envdiscover.Var) []string {
	content, err := os.ReadFile(filepath.Join(workspace, composeFile))
	if err != nil {
		return nil
	}
	withDefaults := make(map[string]struct{})
	for _, item := range envVars {
		if item.DefaultValue != nil {
			withDefaults[item.Key] = struct{}{}
		}
	}
	seen := make(map[string]struct{})
	for _, match := range composeEnvMatches(string(content)) {
		key := match.key
		operator := match.operator
		if operator == "-" || operator == ":-" {
			continue
		}
		if _, ok := withDefaults[key]; ok {
			continue
		}
		seen[key] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func int32FromAny(value any) int32 {
	switch typed := value.(type) {
	case float64:
		return int32(typed)
	case int:
		return int32(typed)
	case string:
		return int32FromString(typed)
	default:
		return 0
	}
}

func int32FromString(value string) int32 {
	value = trimPortValue(value)
	if strings.Contains(value, "/") {
		value = strings.SplitN(value, "/", 2)[0]
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 || parsed > 65535 {
		return 0
	}
	return int32(parsed)
}

func sortComposeIssues(issues []ComposeIssue) {
	rank := map[string]int{"error": 0, "warning": 1, "info": 2}
	sort.SliceStable(issues, func(i, j int) bool {
		left := rank[issues[i].Severity]
		right := rank[issues[j].Severity]
		if left != right {
			return left < right
		}
		if issues[i].Service == nil || issues[j].Service == nil {
			return issues[i].Code < issues[j].Code
		}
		if *issues[i].Service != *issues[j].Service {
			return *issues[i].Service < *issues[j].Service
		}
		return issues[i].Code < issues[j].Code
	})
}
