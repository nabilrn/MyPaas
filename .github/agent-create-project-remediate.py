from pathlib import Path


def replace_once(path: str, old: str, new: str):
    p = Path(path)
    text = p.read_text()
    if old not in text:
        raise SystemExit(f"missing replacement anchor in {path}: {old[:140]!r}")
    p.write_text(text.replace(old, new, 1))


# Static frontend classification must run before optional Nixpacks enrichment.
replace_once(
    "backend/internal/project/service.go",
    '''\t// Vibecoder Fallback: use nixpacks plan
\tplan, _ := nixpacks.PlanWorkspace(ctx, workspace)
\tif plan != nil && isStaticSPA(workspace) {
\t\treturn DetectResult{DeployMode: "static", Branch: branch, HasDockerfile: false, EnvVars: envVars, AppPort: 80, Tree: tree, TreeTruncated: treeTruncated}, nil
\t}

\t// If it's not a static SPA (or if nixpacks failed), reject it and provide the AI prompt
''',
    '''\t// Known static frontend frameworks are classified directly from repository signals.
\t// Nixpacks remains optional enrichment for unknown runtime projects; it is not a
\t// prerequisite for recognizing a static build.
\tif isStaticSPA(workspace) {
\t\treturn DetectResult{DeployMode: "static", Branch: branch, HasDockerfile: false, EnvVars: envVars, AppPort: 80, Tree: tree, TreeTruncated: treeTruncated}, nil
\t}

\tplan, _ := nixpacks.PlanWorkspace(ctx, workspace)

\t// If it is not a known static frontend, reject it and provide the AI prompt.
''',
)
replace_once(
    "backend/internal/project/service.go",
    '''func isStaticSPA(workspace string) bool {
\tb, err := os.ReadFile(filepath.Join(workspace, "package.json"))
\tif err != nil {
\t\treturn false
\t}
\tcontent := string(b)

\t// If it has SSR/Backend frameworks, it's not an SPA
\tif strings.Contains(content, `"next"`) || strings.Contains(content, `"nuxt"`) || strings.Contains(content, `"@nestjs/core"`) {
\t\treturn false
\t}

\t// If it has SPA frameworks/bundlers
\tif strings.Contains(content, `"vite"`) || strings.Contains(content, `"react-scripts"`) || strings.Contains(content, `"@sveltejs/adapter-static"`) || strings.Contains(content, `"astro"`) || strings.Contains(content, `"vue-cli-service"`) {
\t\treturn true
\t}

\treturn false
}
''',
    '''func isStaticSPA(workspace string) bool {
\treturn isStaticFrontend(workspace)
}
''',
)

# Agent assistance is a first-class Create Project action, not buried in Advanced.
replace_once(
    "frontend/src/routes/projects/new/+page.svelte",
    "\timport { Check, ChevronDown, CircleAlert, Copy, Folder, LoaderCircle, Plus, RefreshCw, Rocket, Upload, X } from '@lucide/svelte';",
    "\timport { Bot, Check, ChevronDown, CircleAlert, Copy, Folder, LoaderCircle, Plus, RefreshCw, Rocket, Upload, X } from '@lucide/svelte';",
)
replace_once(
    "frontend/src/routes/projects/new/+page.svelte",
    "\timport ActionButton from '$components/ActionButton.svelte';\n",
    "\timport ActionButton from '$components/ActionButton.svelte';\n\timport ActionLink from '$components/ActionLink.svelte';\n",
)
replace_once(
    "frontend/src/routes/projects/new/+page.svelte",
    '''\t$: actionableHandoffPrompt = backendPromptParts.length > 1
\t\t? backendPromptParts[1].trim()
\t\t: (detectError || composeBlockingIssues.length > 0 ? generatedHandoffPrompt : '');''',
    '''\t$: actionableHandoffPrompt = backendPromptParts.length > 1
\t\t? backendPromptParts[1].trim()
\t\t: form.sourceType === 'git' && (form.deployMode === 'dockerfile' || form.deployMode === 'compose')
\t\t\t? generatedHandoffPrompt
\t\t\t: '';''',
)
replace_once(
    "frontend/src/routes/projects/new/+page.svelte",
    "\t\t\tmode === 'compose' ? 'Validate with the relevant project checks and `docker compose config`.' : 'Validate with the relevant project checks and a production `docker build`.',\n\t\t\t'Do not deploy, push, or commit unless explicitly asked. Finish with the exact MyPaas settings to use.'",
    "\t\t\tmode === 'compose' ? 'Validate with the relevant project checks and `docker compose config`.' : 'Validate with the relevant project checks and a production `docker build`.',\n\t\t\t'If the MyPaaS MCP server is available, use it only for platform inspection, configuration, and deployment after repository changes are ready; MCP does not replace editing the source repository.',\n\t\t\t'Do not deploy, push, or commit unless explicitly asked. Finish with the exact MyPaas settings to use.'",
)

old_advanced_handoff = '''\n\t\t\t\t\t\t{#if actionableHandoffPrompt}
\t\t\t\t\t\t\t<div class="border-l-2 border-gray-300 pl-3 dark:border-gray-700">
\t\t\t\t\t\t\t\t<div class="flex flex-wrap items-center justify-between gap-2">
\t\t\t\t\t\t\t\t\t<div><p class="text-xs font-medium text-gray-700 dark:text-gray-200">Coding-agent handoff</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Available only when repository changes may be required.</p></div>
\t\t\t\t\t\t\t\t\t<ActionButton variant="secondary" size="xs" type="button" on:click={copyHandoffPrompt}><span class="inline-flex items-center gap-1.5"><Copy class="h-3.5 w-3.5" aria-hidden="true" />{copiedHandoffPrompt === actionableHandoffPrompt ? 'Copied' : 'Copy prompt'}</span></ActionButton>
\t\t\t\t\t\t\t\t</div>
\t\t\t\t\t\t\t</div>
\t\t\t\t\t\t{/if}'''
replace_once("frontend/src/routes/projects/new/+page.svelte", old_advanced_handoff, "")

agent_section = '''\t\t\t{#if showSetupSummary && (form.sourceType === 'registry' || actionableHandoffPrompt)}
\t\t\t\t<section class="border-t border-gray-100 p-5 sm:p-6 dark:border-gray-800">
\t\t\t\t\t<div class="flex flex-col gap-4 rounded-md border border-gray-200 px-4 py-4 dark:border-gray-800 sm:flex-row sm:items-center sm:justify-between">
\t\t\t\t\t\t<div class="flex min-w-0 gap-3">
\t\t\t\t\t\t\t<Bot class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
\t\t\t\t\t\t\t<div class="min-w-0">
\t\t\t\t\t\t\t\t<p class="text-sm font-semibold text-gray-950 dark:text-white">Agent assistance</p>
\t\t\t\t\t\t\t\t{#if form.sourceType === 'registry'}
\t\t\t\t\t\t\t\t\t<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">Container images do not need a source-code handoff. Use MyPaaS MCP when you want an agent to inspect, configure, or deploy this platform.</p>
\t\t\t\t\t\t\t\t{:else}
\t\t\t\t\t\t\t\t\t<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">Coding agents change the repository; MyPaaS MCP handles platform operations after the source is ready. The copied prompt includes only environment keys, never secret values.</p>
\t\t\t\t\t\t\t\t{/if}
\t\t\t\t\t\t\t</div>
\t\t\t\t\t\t</div>
\t\t\t\t\t\t<div class="flex shrink-0 flex-wrap gap-2">
\t\t\t\t\t\t\t{#if actionableHandoffPrompt}
\t\t\t\t\t\t\t\t<ActionButton variant="secondary" size="xs" type="button" on:click={copyHandoffPrompt}>
\t\t\t\t\t\t\t\t\t<Copy slot="icon" class="h-3.5 w-3.5" />
\t\t\t\t\t\t\t\t\t{copiedHandoffPrompt === actionableHandoffPrompt ? 'Copied' : 'Copy agent prompt'}
\t\t\t\t\t\t\t\t</ActionButton>
\t\t\t\t\t\t\t{/if}
\t\t\t\t\t\t\t<ActionLink href="/admin/mcp" variant="secondary" size="xs">
\t\t\t\t\t\t\t\t<Bot slot="icon" class="h-3.5 w-3.5" />
\t\t\t\t\t\t\t\tMCP setup
\t\t\t\t\t\t\t</ActionLink>
\t\t\t\t\t\t</div>
\t\t\t\t\t</div>
\t\t\t\t</section>
\t\t\t{/if}

'''
replace_once(
    "frontend/src/routes/projects/new/+page.svelte",
    '\t\t\t<section class="border-t border-gray-100 p-3 dark:border-gray-800 sm:p-4">\n\t\t\t\t<details class="group rounded-md border border-gray-200 bg-gray-50/60 dark:border-gray-800 dark:bg-gray-900/40">',
    agent_section + '\t\t\t<section class="border-t border-gray-100 p-3 dark:border-gray-800 sm:p-4">\n\t\t\t\t<details class="group rounded-md border border-gray-200 bg-gray-50/60 dark:border-gray-800 dark:bg-gray-900/40">',
)
