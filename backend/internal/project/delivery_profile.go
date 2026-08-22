package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	frameworkUnknown   = "unknown"
	frameworkNextJS    = "nextjs"
	frameworkVite      = "vite"
	frameworkAstro     = "astro"
	frameworkSvelteKit = "sveltekit"
	frameworkNuxt      = "nuxt"
	frameworkNestJS    = "nestjs"
	frameworkNodeAPI   = "node-api"

	deliveryGenericContainer = "generic-container"
	deliveryNextStandalone   = "next-standalone"
	deliveryStaticSite       = "static-site"
	deliverySPAStatic        = "spa-static"
	deliveryAPIOnly          = "api-only"
	deliveryCompose          = "compose"
)

type deliveryProfileResult struct {
	Framework string
	Profile   string
	Warnings  []string
}

func detectDeliveryProfile(workspace, deployMode string) deliveryProfileResult {
	framework := detectFramework(workspace)
	result := deliveryProfileResult{
		Framework: framework,
		Profile:   deliveryGenericContainer,
		Warnings:  []string{},
	}

	switch deployMode {
	case "compose":
		result.Profile = deliveryCompose
		result.Warnings = append(result.Warnings, "Compose detected. MyPaaS routes the selected main service; per-service framework optimization is not inferred yet.")
	case "static":
		switch framework {
		case frameworkVite:
			result.Profile = deliverySPAStatic
		case frameworkAstro, frameworkSvelteKit, frameworkUnknown:
			result.Profile = deliveryStaticSite
		default:
			result.Profile = deliveryStaticSite
		}
		result.Warnings = append(result.Warnings, "Static delivery detected. Immutable asset caching, compression, and SPA/history fallback are applied by Caddy.")
	case "dockerfile", "image":
		switch framework {
		case frameworkNextJS:
			result.Profile = deliveryNextStandalone
			result.Warnings = append(result.Warnings, "Next.js runtime detected. Keep /api/* non-cacheable and offload /_next/static/* through Caddy or an upstream CDN.")
		case frameworkNestJS, frameworkNodeAPI:
			result.Profile = deliveryAPIOnly
			result.Warnings = append(result.Warnings, "API runtime detected. Latency depends mostly on application handlers, database calls, and upstream proxy path.")
		default:
			result.Profile = deliveryGenericContainer
			result.Warnings = append(result.Warnings, "No framework-specific delivery profile detected; using generic container routing.")
		}
	default:
		result.Warnings = append(result.Warnings, "Unknown deploy mode; using generic delivery guidance.")
	}

	return result
}

func detectFramework(workspace string) string {
	data, err := os.ReadFile(filepath.Join(workspace, "package.json"))
	if err != nil {
		return frameworkUnknown
	}
	var manifest frontendPackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return frameworkUnknown
	}

	switch {
	case packageHas(manifest, "next"):
		return frameworkNextJS
	case packageHas(manifest, "nuxt"):
		return frameworkNuxt
	case packageHas(manifest, "astro"):
		return frameworkAstro
	case packageHas(manifest, "@sveltejs/kit"):
		return frameworkSvelteKit
	case packageHas(manifest, "vite"):
		return frameworkVite
	case packageHas(manifest, "@nestjs/core"):
		return frameworkNestJS
	case packageHas(manifest, "express") || packageHas(manifest, "fastify") || packageHas(manifest, "hono") || packageHas(manifest, "koa"):
		return frameworkNodeAPI
	}

	for _, script := range manifest.Scripts {
		normalized := strings.ToLower(script)
		if strings.Contains(normalized, "next ") {
			return frameworkNextJS
		}
		if strings.Contains(normalized, "vite") {
			return frameworkVite
		}
	}

	return frameworkUnknown
}
