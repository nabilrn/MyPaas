package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"mypaas/internal/db"
)

const (
	staticBuilderNodeImage = "node:22-alpine"
	// Static builds are untrusted repository workloads. Keep the v0.1 builder
	// ceiling deliberately simple and portable across Docker and Podman's
	// Docker-compatible CLI. The deployment context still provides the outer
	// build timeout.
	staticBuilderMemoryMB = 2048
	staticBuilderCPU      = 2.0
	staticBuilderPIDs     = 512
)

type staticPackageJSON struct {
	PackageManager string            `json:"packageManager"`
	Scripts        map[string]string `json:"scripts"`
}

type staticBuildPlan struct {
	Image          string
	PackageManager string
	Command        string
}

func needsStaticBuild(workspace string) bool {
	pkg, err := readStaticPackageJSON(workspace)
	if err != nil {
		return false
	}
	_, ok := pkg.Scripts["build"]
	return ok
}

func readStaticPackageJSON(workspace string) (staticPackageJSON, error) {
	var pkg staticPackageJSON
	b, err := os.ReadFile(filepath.Join(workspace, "package.json"))
	if err != nil {
		return pkg, err
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		return pkg, fmt.Errorf("decode package.json: %w", err)
	}
	return pkg, nil
}

func staticBuildPlanForWorkspace(workspace string) (staticBuildPlan, error) {
	pkg, err := readStaticPackageJSON(workspace)
	if err != nil {
		return staticBuildPlan{}, fmt.Errorf("read static package metadata: %w", err)
	}
	if _, ok := pkg.Scripts["build"]; !ok {
		return staticBuildPlan{}, fmt.Errorf("package.json does not define a build script")
	}

	manager, version := parsePackageManager(pkg.PackageManager)
	if manager == "" {
		manager = inferPackageManagerFromLockfile(workspace)
	}
	if manager == "" {
		manager = "npm"
	}

	plan := staticBuildPlan{
		Image:          staticBuilderNodeImage,
		PackageManager: manager,
	}

	switch manager {
	case "pnpm":
		// Node 22 still ships Corepack. With packageManager present, Corepack
		// selects the exact pnpm version declared by the project.
		plan.Command = "corepack enable && pnpm install --frozen-lockfile && pnpm run build"
	case "yarn":
		installFlag := "--immutable"
		if strings.HasPrefix(version, "1.") || (version == "" && !fileExists(filepath.Join(workspace, ".yarnrc.yml"))) {
			installFlag = "--frozen-lockfile"
		}
		plan.Command = fmt.Sprintf("corepack enable && yarn install %s && yarn run build", installFlag)
	case "npm":
		install := "npm install"
		if fileExists(filepath.Join(workspace, "package-lock.json")) || fileExists(filepath.Join(workspace, "npm-shrinkwrap.json")) {
			install = "npm ci"
		}
		plan.Command = install + " && npm run build"
	default:
		return staticBuildPlan{}, fmt.Errorf("unsupported static package manager %q; supported package managers are npm, pnpm, and yarn", manager)
	}

	return plan, nil
}

func parsePackageManager(spec string) (string, string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", ""
	}
	name, version, found := strings.Cut(spec, "@")
	if !found {
		return strings.ToLower(strings.TrimSpace(spec)), ""
	}
	return strings.ToLower(strings.TrimSpace(name)), strings.TrimSpace(version)
}

func inferPackageManagerFromLockfile(workspace string) string {
	switch {
	case fileExists(filepath.Join(workspace, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(workspace, "yarn.lock")):
		return "yarn"
	case fileExists(filepath.Join(workspace, "package-lock.json")), fileExists(filepath.Join(workspace, "npm-shrinkwrap.json")):
		return "npm"
	default:
		return ""
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func staticBuilderRunArgs(workspace string, plan staticBuildPlan) []string {
	return []string{
		"run", "--rm",
		"--memory", fmt.Sprintf("%dm", staticBuilderMemoryMB),
		"--cpus", fmt.Sprintf("%.2f", staticBuilderCPU),
		"--pids-limit", fmt.Sprint(staticBuilderPIDs),
		"-v", fmt.Sprintf("%s:/app", workspace),
		"-w", "/app",
		plan.Image,
		"sh", "-c", plan.Command,
	}
}

// buildStaticSPA builds the selected static workspace in an ephemeral Node
// container. The workspace is already scoped to project.BaseDirectory by the
// deployment lifecycle, so package metadata and lockfiles are resolved from
// the selected subdirectory rather than the repository root.
func (s *Service) buildStaticSPA(ctx context.Context, project db.Project, deploymentID uuid.UUID, workspace string, log func(string)) error {
	log("No pre-built static files found. Starting ephemeral static builder...")

	plan, err := staticBuildPlanForWorkspace(workspace)
	if err != nil {
		return fmt.Errorf("prepare static build: %w", err)
	}

	log(fmt.Sprintf("Static builder detected %s; running build in %s...", plan.PackageManager, plan.Image))

	cmd := exec.CommandContext(ctx, "docker", staticBuilderRunArgs(workspace, plan)...)

	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		log(string(out))
	}
	if err != nil {
		return fmt.Errorf("static build failed with %s in %s: %w", plan.PackageManager, plan.Image, err)
	}

	log("Static build completed successfully.")
	return nil
}
