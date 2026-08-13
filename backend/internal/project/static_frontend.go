package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type frontendPackageManifest struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func isStaticFrontend(workspace string) bool {
	data, err := os.ReadFile(filepath.Join(workspace, "package.json"))
	if err != nil {
		return false
	}
	var manifest frontendPackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return false
	}

	if packageHas(manifest, "next") || packageHas(manifest, "nuxt") || packageHas(manifest, "@nestjs/core") {
		return false
	}

	build := strings.ToLower(strings.TrimSpace(manifest.Scripts["build"]))
	if packageHas(manifest, "astro") {
		if !strings.Contains(build, "astro build") {
			return false
		}
		return !astroUsesServerRuntime(workspace, manifest)
	}
	if packageHas(manifest, "@sveltejs/kit") {
		return packageHas(manifest, "@sveltejs/adapter-static") && build != ""
	}
	if packageHas(manifest, "@sveltejs/adapter-static") {
		return build != ""
	}
	if packageHas(manifest, "vite") || packageHas(manifest, "react-scripts") || packageHas(manifest, "vue-cli-service") {
		return build != ""
	}
	return false
}

func packageHas(manifest frontendPackageManifest, name string) bool {
	if _, ok := manifest.Dependencies[name]; ok {
		return true
	}
	_, ok := manifest.DevDependencies[name]
	return ok
}

func astroUsesServerRuntime(workspace string, manifest frontendPackageManifest) bool {
	serverAdapters := []string{"@astrojs/node", "@astrojs/cloudflare", "@astrojs/netlify", "@astrojs/vercel"}
	configNames := []string{"astro.config.mjs", "astro.config.js", "astro.config.ts", "astro.config.cjs"}
	for _, name := range configNames {
		data, err := os.ReadFile(filepath.Join(workspace, name))
		if err != nil {
			continue
		}
		normalized := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "").Replace(strings.ToLower(string(data)))
		if strings.Contains(normalized, "output:'server'") || strings.Contains(normalized, "output:\"server\"") ||
			strings.Contains(normalized, "output:'hybrid'") || strings.Contains(normalized, "output:\"hybrid\"") {
			return true
		}
		if strings.Contains(normalized, "adapter:") {
			for _, adapter := range serverAdapters {
				if packageHas(manifest, adapter) {
					return true
				}
			}
		}
	}
	return false
}
