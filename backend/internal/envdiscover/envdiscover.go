package envdiscover

import (
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
	Key          string  `json:"key"`
	Source       string  `json:"source"`
	Sensitive    bool    `json:"sensitive"`
	DefaultValue *string `json:"defaultValue,omitempty"`
}

type composeVariableMatch struct {
	key          string
	operator     string
	defaultValue string
}

func Discover(workspace, composeFile string) ([]Var, error) {
	found := make(map[string]Var)
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
		vars = append(vars, item)
	}
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Key < vars[j].Key
	})
	return vars, nil
}

func discoverEnvFiles(found map[string]Var, workspace string) error {
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

func discoverEnvFile(found map[string]Var, source, content string) {
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

func discoverCompose(found map[string]Var, source, content string) {
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

func add(found map[string]Var, key, source, defaultValue string) {
	sensitive := isSensitive(key)
	item, ok := found[key]
	if !ok {
		item = Var{Key: key, Source: source, Sensitive: sensitive}
	} else {
		item.Sensitive = item.Sensitive || sensitive
		if !strings.Contains(item.Source, source) {
			item.Source += ", " + source
		}
	}
	if !item.Sensitive && item.DefaultValue == nil && defaultValue != "" {
		value := defaultValue
		item.DefaultValue = &value
	}
	found[key] = item
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
