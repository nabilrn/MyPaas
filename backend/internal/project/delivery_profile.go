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
		result.Warnings = append(result.Warnings, "Compose detected. MyPaaS routes the selected main service and compresses proxied responses; per-service static artifact extraction is not inferred.")
	case "static":
		switch framework {
		case frameworkVite:
			result.Profile = deliverySPAStatic
		default:
			result.Profile = deliveryStaticSite
		}
		result.Warnings = append(result.Warnings, staticDeliveryWarning(framework))
	case "dockerfile", "image":
		switch framework {
		case frameworkNextJS:
			result.Profile = deliveryNextStandalone
			result.Warnings = append(result.Warnings, "Next.js runtime detected. Caddy compresses proxied responses; preserve Next cache semantics and treat /_next/static/* as immutable only when those build artifacts are explicitly published outside the image.")
		case frameworkNuxt:
			result.Profile = deliveryGenericContainer
			result.Warnings = append(result.Warnings, "Nuxt SSR detected. Caddy compresses proxied responses; /_nuxt/* is the framework-owned static namespace, but direct Caddy file delivery requires an explicit published artifact rather than guessing inside the image.")
		case frameworkSvelteKit:
			result.Profile = deliveryGenericContainer
			result.Warnings = append(result.Warnings, "SvelteKit Node runtime detected. Caddy handles proxy compression; existing .br/.gz sidecars are preserved when static output is published, and /_app/immutable/* is safe for immutable caching only in a published static tree.")
		case frameworkNestJS, frameworkNodeAPI:
			result.Profile = deliveryAPIOnly
			result.Warnings = append(result.Warnings, "API runtime detected. Caddy compresses eligible responses; API caching remains application-controlled and database/handler latency is not masked by a generic cache rule.")
		default:
			result.Profile = deliveryGenericContainer
			result.Warnings = append(result.Warnings, "No framework-specific delivery profile detected. Caddy provides reverse-proxy compression without inventing cache semantics for the application.")
		}
	default:
		result.Warnings = append(result.Warnings, "Unknown deploy mode; using generic delivery guidance.")
	}

	return result
}

func staticDeliveryWarning(framework string) string {
	switch framework {
	case frameworkVite:
		return "Vite static delivery detected. MyPaaS serves the generated tree directly with Caddy, revalidates unhashed files, and gives default hash-named /assets/*-*.* build artifacts immutable caching."
	case frameworkAstro:
		return "Astro static delivery detected. MyPaaS serves the generated tree directly with Caddy, revalidates HTML/unhashed files, and gives /_astro/* immutable caching."
	case frameworkNuxt:
		return "Nuxt static delivery detected. MyPaaS serves the generated tree directly with Caddy, revalidates HTML/unhashed files, and gives /_nuxt/* immutable caching."
	case frameworkSvelteKit:
		return "SvelteKit static delivery detected. MyPaaS serves the generated tree directly with Caddy, revalidates HTML/unhashed files, gives /_app/immutable/* immutable caching, and can serve existing Brotli/gzip sidecars."
	default:
		return "Static delivery detected. MyPaaS serves files directly with Caddy, revalidates unhashed content, and only applies immutable caching to known fingerprinted framework asset namespaces."
	}
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
		if strings.Contains(normalized, "nuxt") {
			return frameworkNuxt
		}
		if strings.Contains(normalized, "vite") {
			return frameworkVite
		}
	}

	return frameworkUnknown
}
