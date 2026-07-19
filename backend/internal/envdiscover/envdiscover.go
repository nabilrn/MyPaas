package envdiscover

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	keyPattern          = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	composeVariableExpr = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?:(:?[-?])([^}]*))?\}`)
	sensitiveTokens     = []string{"SECRET", "TOKEN", "PASSWORD", "PASS", "KEY", "DATABASE_URL", "DSN", "PRIVATE"}
	envFiles            = []string{".env.example", ".env.sample", ".env.template", ".env.local.example"}
	envFileNames        = map[string]struct{}{}
	skippedDirs         = map[string]struct{}{
		".cache":       {},
		".git":         {},
		".next":        {},
		".svelte-kit":  {},
		"build":        {},
		"coverage":     {},
		"dist":         {},
		"node_modules": {},
		"target":       {},
		"vendor":       {},
	}
)

const (
	maxDiscoverDepth = 4
	maxEnvFiles      = 24
)

func init() {
	for _, name := range envFiles {
		envFileNames[name] = struct{}{}
	}
}

type Var struct {
	Key          string   `json:"key"`
	Source       string   `json:"source"`
	Sensitive    bool     `json:"sensitive"`
	DefaultValue *string  `json:"defaultValue,omitempty"`
	Services     []string `json:"services,omitempty"`
	Conflict     *Conflict `json:"conflict,omitempty"`
}

// Conflict describes a situation where the same env var key has different
// default values across different service .env.example files. The user must
// decide which value to use.
type Conflict struct {
	// Values maps each default value to the services that declare it.
	Values []ConflictValue `json:"values"`
}

type ConflictValue struct {
	Value   string   `json:"value"`
	Sources []string `json:"sources"`
}

type composeVariableMatch struct {
	key          string
	operator     string
	defaultValue string
}

// ServiceEnvFile describes an env_file entry from a compose service config.
type ServiceEnvFile struct {
	Service  string
	Path     string
	Required bool
}

func Discover(workspace, composeFile string) ([]Var, error) {
	found := make(map[string]*varEntry)
	if err := discoverEnvFiles(found, workspace); err != nil {
		return nil, err
	}

	if strings.TrimSpace(composeFile) != "" {
		path := filepath.Join(workspace, composeFile)
		content, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			discoverCompose(found, composeFile, string(content))
		}
	}

	vars := make([]Var, 0, len(found))
	for _, item := range found {
		vars = append(vars, item.toVar())
	}
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Key < vars[j].Key
	})
	return vars, nil
}

// AttributeServicesFromConfig links discovered env vars to compose services
// using the JSON config output by `docker compose config --format json`. This
// requires docker compose to be available, so it's called separately from
// Discover (which only reads files). The caller should run docker compose
// config first, then pass the JSON here.
func AttributeServicesFromConfig(vars []Var, configJSON []byte) []Var {
	serviceEnvKeys := parseServiceEnvKeysFromJSON(configJSON)
	if len(serviceEnvKeys) == 0 {
		return vars
	}
	byKey := make(map[string]*Var, len(vars))
	for i := range vars {
		byKey[vars[i].Key] = &vars[i]
	}
	for key, services := range serviceEnvKeys {
		entry, ok := byKey[key]
		if !ok {
			continue
		}
		if entry.Services == nil {
			entry.Services = make([]string, 0)
		}
		for _, svc := range services {
			if !containsString(entry.Services, svc) {
				entry.Services = append(entry.Services, svc)
			}
		}
		sort.Strings(entry.Services)
	}
	return vars
}

// parseServiceEnvKeysFromJSON extracts a map of env var key → list of
// services that reference it, from the docker compose config JSON.
func parseServiceEnvKeysFromJSON(configJSON []byte) map[string][]string {
	var doc struct {
		Services map[string]struct {
			Environment json.RawMessage `json:"environment"`
		} `json:"services"`
	}
	if err := json.Unmarshal(configJSON, &doc); err != nil {
		return nil
	}
	keyToServices := make(map[string][]string)
	addKey := func(key, service string) {
		key = strings.TrimSpace(key)
		if key == "" {
			return
		}
		for _, existing := range keyToServices[key] {
			if existing == service {
				return
			}
		}
		keyToServices[key] = append(keyToServices[key], service)
	}
	for service, spec := range doc.Services {
		for key := range parseComposeEnvironmentMap(spec.Environment) {
			addKey(key, service)
		}
		for _, item := range parseComposeEnvironmentList(spec.Environment) {
			key, _, ok := strings.Cut(item, "=")
			if ok {
				addKey(key, service)
			}
		}
	}
	return keyToServices
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

// varEntry is the internal accumulator for discovered vars. It tracks
// per-service default values so conflicts can be detected.
type varEntry struct {
	key          string
	sources      []string
	sensitive    bool
	defaultValue *string
	// serviceDefaults maps service name → default value declared in that
	// service's env file. Used for conflict detection.
	serviceDefaults map[string]string
	// services is the set of compose services that reference this var.
	services map[string]struct{}
}

func (e *varEntry) toVar() Var {
	out := Var{
		Key:          e.key,
		Source:       strings.Join(e.sources, ", "),
		Sensitive:    e.sensitive,
		DefaultValue: e.defaultValue,
	}
	if len(e.services) > 0 {
		out.Services = sortedKeys(e.services)
	}
	if conflict := e.detectConflict(); conflict != nil {
		out.Conflict = conflict
	}
	return out
}

func (e *varEntry) detectConflict() *Conflict {
	if len(e.serviceDefaults) < 2 {
		return nil
	}
	byValue := make(map[string][]string)
	for svc, val := range e.serviceDefaults {
		byValue[val] = append(byValue[val], svc)
	}
	if len(byValue) < 2 {
		return nil
	}
	values := make([]ConflictValue, 0, len(byValue))
	for val, svcs := range byValue {
		sort.Strings(svcs)
		values = append(values, ConflictValue{Value: val, Sources: svcs})
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].Value < values[j].Value
	})
	return &Conflict{Values: values}
}

func (e *varEntry) addSource(source string) {
	for _, existing := range e.sources {
		if existing == source {
			return
		}
	}
	e.sources = append(e.sources, source)
}

func (e *varEntry) addService(service string) {
	if e.services == nil {
		e.services = make(map[string]struct{})
	}
	e.services[service] = struct{}{}
}

func (e *varEntry) addServiceDefault(service, value string) {
	if e.serviceDefaults == nil {
		e.serviceDefaults = make(map[string]string)
	}
	if _, ok := e.serviceDefaults[service]; !ok {
		e.serviceDefaults[service] = value
	}
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func discoverEnvFiles(found map[string]*varEntry, workspace string) error {
	count := 0
	return filepath.WalkDir(workspace, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == workspace {
			return nil
		}
		rel, err := filepath.Rel(workspace, path)
		if err != nil {
			return err
		}
		depth := pathDepth(rel)
		if entry.IsDir() {
			if shouldSkipDir(entry.Name()) || depth >= maxDiscoverDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if count >= maxEnvFiles {
			return nil
		}
		if _, ok := envFileNames[entry.Name()]; !ok {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		discoverEnvFile(found, filepath.ToSlash(rel), string(content))
		count++
		return nil
	})
}

func discoverEnvFile(found map[string]*varEntry, source, content string) {
	for _, line := range strings.Split(strings.ReplaceAll(strings.TrimPrefix(content, "\ufeff"), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if !keyPattern.MatchString(key) {
			continue
		}
		value = strings.TrimSpace(stripInlineComment(value))
		add(found, key, source, unquote(value))
	}
}

func discoverCompose(found map[string]*varEntry, source, content string) {
	for _, match := range composeVariableMatches(content) {
		key := match.key
		if !keyPattern.MatchString(key) {
			continue
		}
		defaultValue := ""
		if match.operator == "-" || match.operator == ":-" {
			defaultValue = strings.TrimSpace(match.defaultValue)
		}
		add(found, key, source, defaultValue)
	}
}

// parseEnvFileEntries extracts file paths from a compose `env_file:` field,
// which can be a string, list of strings, or list of objects with `path`.
func parseEnvFileEntries(raw json.RawMessage) []ServiceEnvFile {
	var entries []ServiceEnvFile
	if len(raw) == 0 || string(raw) == "null" {
		return entries
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		entries = append(entries, ServiceEnvFile{Path: asString, Required: true})
		return entries
	}
	var asList []string
	if err := json.Unmarshal(raw, &asList); err == nil {
		for _, p := range asList {
			entries = append(entries, ServiceEnvFile{Path: p, Required: true})
		}
		return entries
	}
	var asObjList []struct {
		Path     string `json:"path"`
		Required *bool  `json:"required"`
	}
	if err := json.Unmarshal(raw, &asObjList); err == nil {
		for _, obj := range asObjList {
			required := true
			if obj.Required != nil {
				required = *obj.Required
			}
			entries = append(entries, ServiceEnvFile{Path: obj.Path, Required: required})
		}
	}
	return entries
}

func parseComposeEnvironmentMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err == nil {
		return asMap
	}
	return nil
}

func parseComposeEnvironmentList(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var asList []string
	if err := json.Unmarshal(raw, &asList); err == nil {
		return asList
	}
	return nil
}

func composeVariableMatches(content string) []composeVariableMatch {
	indexes := composeVariableExpr.FindAllStringSubmatchIndex(content, -1)
	matches := make([]composeVariableMatch, 0, len(indexes))
	for _, index := range indexes {
		if len(index) < 8 {
			continue
		}
		start := index[0]
		if start > 0 && content[start-1] == '$' {
			continue
		}
		item := composeVariableMatch{key: content[index[2]:index[3]]}
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

func add(found map[string]*varEntry, key, source, defaultValue string) {
	sensitive := isSensitive(key)
	entry, ok := found[key]
	if !ok {
		entry = &varEntry{key: key, sensitive: sensitive}
		found[key] = entry
	} else {
		entry.sensitive = entry.sensitive || sensitive
	}
	entry.addSource(source)
	if !entry.sensitive && entry.defaultValue == nil && defaultValue != "" {
		value := defaultValue
		entry.defaultValue = &value
	}
}

// AddServiceDefault records that a specific service expects a specific
// default value for this var. Used for conflict detection.
func AddServiceDefault(found map[string]*varEntry, key, service, defaultValue string) {
	entry, ok := found[key]
	if !ok {
		return
	}
	entry.addServiceDefault(service, defaultValue)
}

func isSensitive(key string) bool {
	upper := strings.ToUpper(key)
	for _, token := range sensitiveTokens {
		if strings.Contains(upper, token) {
			return true
		}
	}
	return false
}

func pathDepth(rel string) int {
	depth := 0
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part != "" {
			depth++
		}
	}
	return depth
}

func shouldSkipDir(name string) bool {
	_, ok := skippedDirs[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func stripInlineComment(value string) string {
	inSingle := false
	inDouble := false
	escaped := false
	for i, ch := range value {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if ch == '#' && !inSingle && !inDouble && (i == 0 || value[i-1] == ' ' || value[i-1] == '\t') {
			return strings.TrimSpace(value[:i])
		}
	}
	return value
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

// ParseComposeEnvFiles extracts env_file entries from a compose config JSON
// (as output by `docker compose config --format json`). Returns one entry
// per service-envfile combination.
func ParseComposeEnvFiles(configJSON []byte) []ServiceEnvFile {
	var doc struct {
		Services map[string]struct {
			EnvFile json.RawMessage `json:"env_file"`
		} `json:"services"`
	}
	if err := json.Unmarshal(configJSON, &doc); err != nil {
		return nil
	}
	var entries []ServiceEnvFile
	for service, spec := range doc.Services {
		for _, ef := range parseEnvFileEntries(spec.EnvFile) {
			ef.Service = service
			entries = append(entries, ef)
		}
	}
	return entries
}

// FormatEnvFile renders a key→value map as a .env file content string.
// Used by the deployment engine to auto-generate per-service .env files.
func FormatEnvFile(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(values[key])
		b.WriteString("\n")
	}
	return b.String()
}

// GenerateEnvFromTemplate reads a .env.example template file, substitutes
// values from the provided overrides map, and returns the content for the
// target .env file. Keys not in overrides keep their default from the
// template. Keys in overrides that are not in the template are appended.
func GenerateEnvFromTemplate(templatePath string, overrides map[string]string) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("read env template %s: %w", templatePath, err)
	}
	return fillEnvTemplate(string(content), overrides), nil
}

func fillEnvTemplate(template string, overrides map[string]string) string {
	var b strings.Builder
	seen := make(map[string]struct{})
	for _, line := range strings.Split(strings.ReplaceAll(strings.TrimPrefix(template, "\ufeff"), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			b.WriteString(line)
			b.WriteString("\n")
			continue
		}
		cleaned := strings.TrimPrefix(trimmed, "export ")
		key, _, ok := strings.Cut(cleaned, "=")
		if !ok {
			b.WriteString(line)
			b.WriteString("\n")
			continue
		}
		key = strings.TrimSpace(key)
		if !keyPattern.MatchString(key) {
			b.WriteString(line)
			b.WriteString("\n")
			continue
		}
		seen[key] = struct{}{}
		if value, hasOverride := overrides[key]; hasOverride {
			b.WriteString(key)
			b.WriteString("=")
			b.WriteString(value)
			b.WriteString("\n")
		} else {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	// Append any override keys that weren't in the template.
	overrideKeys := make([]string, 0, len(overrides))
	for key := range overrides {
		if _, wasSeen := seen[key]; !wasSeen {
			overrideKeys = append(overrideKeys, key)
		}
	}
	sort.Strings(overrideKeys)
	for _, key := range overrideKeys {
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(overrides[key])
		b.WriteString("\n")
	}
	return b.String()
}
