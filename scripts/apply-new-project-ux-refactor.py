from pathlib import Path

path = Path("frontend/src/routes/projects/new/+page.svelte")
text = path.read_text()


def replace_once(old: str, new: str, label: str) -> None:
    global text
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly one match, found {count}")
    text = text.replace(old, new, 1)


replace_once(
    "\timport { resolveProjectAppPort } from '$lib/validation/project';",
    "\timport { projectCreationReadiness, resolveProjectAppPort, suggestProjectName } from '$lib/validation/project';",
    "validation imports",
)

replace_once(
    "\tlet appPortSource: PortSource = 'unresolved';\n\tlet envFileInput: HTMLInputElement | null = null;",
    "\tlet appPortSource: PortSource = 'unresolved';\n\tlet projectNameTouched = false;\n\tlet deployModeManual = false;\n\tlet envFileInput: HTMLInputElement | null = null;",
    "local UX state",
)

old_readiness = """\t$: runtimePortMissing = form.deployMode !== 'static' && form.deployMode !== 'auto' && !form.appPort.trim();
\t$: portToServiceMap = buildPortToServiceMap(composePlan?.services ?? []);
\t$: localhostEnvWarnings = detectLocalhostInEnvDrafts(envDrafts, portToServiceMap);
\t$: sourceReady = form.sourceType === 'registry'
\t\t? Boolean(form.imageRef.trim())
\t\t: Boolean(form.repoUrl.trim() && form.branch.trim() && repositoryInspectionCurrent);
\t$: currentRepoInspectKey = [form.repoUrl.trim(), form.branch.trim(), form.baseDirectory.trim()].join('\\n');
\t$: repositoryInspectionCurrent = Boolean(
\t\tform.repoUrl.trim()
\t\t&& form.branch.trim()
\t\t&& lastRepoInspectKey === currentRepoInspectKey
\t\t&& !repoInspectError
\t);
\t$: canSubmit = Boolean(
\t\tform.name.trim()
\t\t&& sourceReady
\t\t&& !runtimePortMissing
\t\t&& !composeDisabledReason
\t\t&& !submitting
\t\t&& !detecting
\t\t&& !inspectingRepo
\t);
\t$: createDisabledReason = !form.name.trim()
\t\t? 'Project name is required'
\t\t: form.sourceType === 'registry' && !form.imageRef.trim()
\t\t\t? 'Container image is required'
\t\t\t: form.sourceType === 'git' && !form.repoUrl.trim()
\t\t\t\t? 'Repository URL is required'
\t\t\t\t: form.sourceType === 'git' && !form.branch.trim()
\t\t\t\t\t? 'Branch is required'
\t\t\t\t\t: form.sourceType === 'git' && repoInspectError
\t\t\t\t\t\t? repoInspectError
\t\t\t\t\t\t: form.sourceType === 'git' && !repositoryInspectionCurrent
\t\t\t\t\t\t\t? 'Repository validation is required for the current branch and base directory'
\t\t\t\t\t\t\t: runtimePortMissing
\t\t\t\t\t\t\t\t? form.sourceType === 'registry'
\t\t\t\t\t\t\t\t\t? 'Container port is required for registry images. Set it in Advanced runtime settings.'
\t\t\t\t\t\t\t\t\t: 'Application port has not been detected. Run Detect runtime or set an Advanced override.'
\t\t\t\t\t\t\t\t: composeDisabledReason
\t\t\t\t\t\t\t\t? composeDisabledReason
\t\t\t\t\t\t\t\t: inspectingRepo
\t\t\t\t\t\t\t\t\t? 'Repository branches are loading'
\t\t\t\t\t\t\t\t\t: detecting
\t\t\t\t\t\t\t\t\t\t? 'Repository detection is running'
\t\t\t\t\t\t\t\t\t\t: submitting
\t\t\t\t\t\t\t\t\t\t\t? 'Project creation is running'
\t\t\t\t\t\t\t\t\t\t\t: '';
\t$: reviewStateLabel = canSubmit ? 'Ready to create' : createDisabledReason || 'Complete required fields';"""

new_readiness = """\t$: portToServiceMap = buildPortToServiceMap(composePlan?.services ?? []);
\t$: localhostEnvWarnings = detectLocalhostInEnvDrafts(envDrafts, portToServiceMap);
\t$: currentRepoInspectKey = [form.repoUrl.trim(), form.branch.trim(), form.baseDirectory.trim()].join('\\n');
\t$: repositoryInspectionCurrent = Boolean(
\t\tform.repoUrl.trim()
\t\t&& form.branch.trim()
\t\t&& lastRepoInspectKey === currentRepoInspectKey
\t\t&& !repoInspectError
\t);
\t$: sourceReady = form.sourceType === 'registry'
\t\t? Boolean(form.imageRef.trim())
\t\t: Boolean(form.repoUrl.trim() && form.branch.trim() && repositoryInspectionCurrent);
\t$: creationReadiness = projectCreationReadiness({
\t\tname: form.name,
\t\tsourceType: form.sourceType,
\t\tsourceReady,
\t\tdeployMode: form.deployMode,
\t\tappPort: form.appPort,
\t\tcomposeDisabledReason,
\t\tbusy: submitting || detecting || inspectingRepo
\t});
\t$: canSubmit = creationReadiness.ready;
\t$: createDisabledReason = form.sourceType === 'git' && !form.repoUrl.trim()
\t\t? 'Repository URL is required'
\t\t: form.sourceType === 'git' && !form.branch.trim() && form.repoUrl.trim()
\t\t\t? 'Select a branch after repository validation'
\t\t\t: form.sourceType === 'git' && repoInspectError
\t\t\t\t? repoInspectError
\t\t\t\t: creationReadiness.reason;
\t$: reviewStateLabel = creationReadiness.state;"""
replace_once(old_readiness, new_readiness, "readiness state")

replace_once("'Ready for detection'", "'Ready for automatic analysis'", "detection label")
replace_once(
    "'Run detection to fill runtime, container port, service, and discovered environment defaults.'",
    "'MyPaas analyzes runtime, container port, services, and environment defaults automatically.'",
    "detection body",
)

replace_once(
    "\t\tform.sourceType = sourceType;\n\t\terror = '';",
    "\t\tform.sourceType = sourceType;\n\t\tdeployModeManual = false;\n\t\terror = '';",
    "source manual reset",
)
replace_once(
    "\tfunction chooseDeployMode(mode: DeployModeChoice) {\n\t\tform.deployMode = mode;",
    "\tfunction chooseDeployMode(mode: DeployModeChoice, manual = true) {\n\t\tdeployModeManual = manual && mode !== 'auto';\n\t\tform.deployMode = mode;",
    "deploy mode manual state",
)
replace_once(
    "\t\tchooseDeployMode(detected.deployMode);",
    "\t\tchooseDeployMode(detected.deployMode, false);",
    "detected mode application",
)
replace_once(
    "\t\tdetectedServices = [];\n\t\tlastRepoInspectKey = '';\n\t}",
    "\t\tdetectedServices = [];\n\t\tlastRepoInspectKey = '';\n\t\tif (!deployModeManual && form.sourceType === 'git') {\n\t\t\tform.deployMode = 'auto';\n\t\t\tform.mainService = '';\n\t\t\tif (appPortSource !== 'manual') {\n\t\t\t\tform.appPort = '';\n\t\t\t\tappPortSource = 'unresolved';\n\t\t\t}\n\t\t}\n\t}",
    "clear stale detected state",
)

replace_once(
    """\tfunction handleRepoUrlInput(event: Event) {
\t\tconst value = (event.currentTarget as HTMLInputElement).value;
\t\tif (value === form.repoUrl) return;
\t\tform.repoUrl = value;
\t\tform.branch = '';
\t\tresetRepositoryInspection();
\t\tscheduleRepositoryInspection();
\t}

\tfunction handleBaseDirectoryInput(event: Event) {""",
    """\tfunction handleNameInput(event: Event) {
\t\tprojectNameTouched = true;
\t\tform.name = (event.currentTarget as HTMLInputElement).value;
\t}

\tfunction handleRepoUrlInput(event: Event) {
\t\tconst value = (event.currentTarget as HTMLInputElement).value;
\t\tif (value === form.repoUrl) return;
\t\tform.repoUrl = value;
\t\tform.branch = '';
\t\tdeployModeManual = false;
\t\tif (!projectNameTouched) form.name = suggestProjectName(value);
\t\tresetRepositoryInspection();
\t\tscheduleRepositoryInspection();
\t}

\tfunction handleImageRefInput(event: Event) {
\t\tconst value = (event.currentTarget as HTMLInputElement).value;
\t\tform.imageRef = value;
\t\tif (!projectNameTouched) form.name = suggestProjectName(value);
\t}

\tfunction handleBaseDirectoryInput(event: Event) {""",
    "source handlers",
)

replace_once(
    """\t\t\tlastRepoInspectKey = repositoryInspectionKey();
\t\t\tif (showToast) {
\t\t\t\ttoast.success('Repository validated');
\t\t\t}
\t\t\treturn inspection;""",
    """\t\t\tlastRepoInspectKey = repositoryInspectionKey();
\t\t\tif (showToast) {
\t\t\t\ttoast.success('Repository validated');
\t\t\t}
\t\t\tif (!deployModeManual) {
\t\t\t\tsetTimeout(() => {
\t\t\t\t\tif (detecting || deployModeManual || lastRepoInspectKey !== repositoryInspectionKey()) return;
\t\t\t\t\tvoid handleDetectMode(false).catch(() => undefined);
\t\t\t\t}, 0);
\t\t\t}
\t\t\treturn inspection;""",
    "automatic runtime analysis",
)

replace_once(
    """\t\t\t\t\t<div>
\t\t\t\t\t\t<label class=\"mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300\" for=\"name\">Project name</label>
\t\t\t\t\t\t<input id=\"name\" type=\"text\" bind:value={form.name} placeholder=\"my-app\" class=\"field w-full\" />
\t\t\t\t\t</div>""",
    """\t\t\t\t\t<div>
\t\t\t\t\t\t<label class=\"mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300\" for=\"name\">Project name</label>
\t\t\t\t\t\t<input id=\"name\" type=\"text\" value={form.name} on:input={handleNameInput} placeholder=\"my-app\" class=\"field w-full\" />
\t\t\t\t\t\t<p class=\"mt-1 text-[11px] text-gray-500 dark:text-gray-400\">Suggested from the source until you edit it. Route preview: <span class=\"font-mono\">{previewHost}</span>.</p>
\t\t\t\t\t</div>""",
    "project name field",
)

old_base = """\t\t\t\t\t\t<div>
\t\t\t\t\t\t\t<label class=\"mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300\" for=\"baseDirectory\">Base directory</label>
\t\t\t\t\t\t\t<input
\t\t\t\t\t\t\t\tid=\"baseDirectory\"
\t\t\t\t\t\t\t\ttype=\"text\"
\t\t\t\t\t\t\t\tvalue={form.baseDirectory}
\t\t\t\t\t\t\t\tplaceholder=\"/\"
\t\t\t\t\t\t\t\tclass=\"field w-full font-mono\"
\t\t\t\t\t\t\t\ton:input={handleBaseDirectoryInput}
\t\t\t\t\t\t\t\ton:blur={() => void inspectRepository(false).catch(() => undefined)}
\t\t\t\t\t\t\t/>
\t\t\t\t\t\t\t<p class=\"mt-1 text-[11px] text-gray-500\">Deploy from a specific subdirectory. E.g. <code>frontend</code> or <code>backend/api</code>.</p>
\t\t\t\t\t\t</div>"""
new_base = """\t\t\t\t\t\t<details class=\"rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40\">
\t\t\t\t\t\t\t<summary class=\"cursor-pointer text-sm font-medium text-gray-950 dark:text-white\">Advanced source settings</summary>
\t\t\t\t\t\t\t<div class=\"mt-3\">
\t\t\t\t\t\t\t\t<label class=\"mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300\" for=\"baseDirectory\">Project directory</label>
\t\t\t\t\t\t\t\t<input
\t\t\t\t\t\t\t\t\tid=\"baseDirectory\"
\t\t\t\t\t\t\t\t\ttype=\"text\"
\t\t\t\t\t\t\t\t\tvalue={form.baseDirectory}
\t\t\t\t\t\t\t\t\tplaceholder=\"Leave blank for repository root\"
\t\t\t\t\t\t\t\t\tclass=\"field w-full font-mono\"
\t\t\t\t\t\t\t\t\ton:input={handleBaseDirectoryInput}
\t\t\t\t\t\t\t\t\ton:blur={() => void inspectRepository(false).catch(() => undefined)}
\t\t\t\t\t\t\t\t/>
\t\t\t\t\t\t\t\t<p class=\"mt-1 text-[11px] text-gray-500\">Only set this for monorepos. Use <code>frontend</code> or <code>apps/api</code>; blank means repository root.</p>
\t\t\t\t\t\t\t</div>
\t\t\t\t\t\t</details>"""
replace_once(old_base, new_base, "advanced source field")

replace_once(
    "<input id=\"imageRef\" type=\"text\" bind:value={form.imageRef} placeholder=\"ghcr.io/example/my-api:v1.4.0\" class=\"field w-full font-mono\" autocomplete=\"off\" />",
    "<input id=\"imageRef\" type=\"text\" value={form.imageRef} on:input={handleImageRefInput} placeholder=\"ghcr.io/example/my-api:v1.4.0\" class=\"field w-full font-mono\" autocomplete=\"off\" />",
    "image source input",
)

replace_once("Detect runtime", "Re-analyze", "runtime action label")
replace_once(
    "\t\t\t\t\t\t<SegmentedChoice label=\"Deployment mode\" value={form.deployMode} options={deployModeOptions} on:change={(event) => chooseDeployMode(event.detail as DeployModeChoice)} />",
    """\t\t\t\t\t\t<div class=\"soft-panel p-3 text-sm\">
\t\t\t\t\t\t\t<p class=\"font-medium text-gray-950 dark:text-white\">Deployment runtime</p>
\t\t\t\t\t\t\t<p class=\"mt-1 text-sm text-gray-950 dark:text-white\">{form.deployMode === 'auto' ? 'Analyzing automatically…' : form.deployMode === 'dockerfile' ? 'Dockerfile' : form.deployMode === 'compose' ? 'Docker Compose' : 'Static site'}</p>
\t\t\t\t\t\t\t<p class=\"mt-0.5 text-xs text-gray-500 dark:text-gray-400\">MyPaas uses the repository as the source of truth. Override only when detection is wrong.</p>
\t\t\t\t\t\t</div>""",
    "runtime result summary",
)

replace_once(
    "\t\t\t\t{#if form.deployMode !== 'static'}\n\t\t\t\t\t<details class=\"rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40\" open={form.sourceType === 'registry' && !form.appPort}>\n\t\t\t\t\t\t<summary class=\"cursor-pointer text-sm font-medium text-gray-950 dark:text-white\">Advanced runtime settings</summary>",
    """\t\t\t\t{#if form.sourceType === 'git' || form.deployMode !== 'static'}
\t\t\t\t\t<details class=\"rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40\" open={form.sourceType === 'registry' && !form.appPort}>
\t\t\t\t\t\t<summary class=\"cursor-pointer text-sm font-medium text-gray-950 dark:text-white\">Advanced runtime settings</summary>
\t\t\t\t\t\t{#if form.sourceType === 'git'}
\t\t\t\t\t\t\t<div class=\"mt-3\">
\t\t\t\t\t\t\t\t<SegmentedChoice label=\"Deployment mode override\" value={form.deployMode} options={deployModeOptions} on:change={(event) => chooseDeployMode(event.detail as DeployModeChoice)} />
\t\t\t\t\t\t\t</div>
\t\t\t\t\t\t{/if}""",
    "advanced runtime mode",
)

replace_once(
    "\t\t\t\t\t\t<div class=\"mt-3 max-w-sm\">\n\t\t\t\t\t\t\t<label class=\"mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300\" for=\"appPort\">Container port override</label>",
    "\t\t\t\t\t\t{#if form.deployMode !== 'static'}\n\t\t\t\t\t\t<div class=\"mt-3 max-w-sm\">\n\t\t\t\t\t\t\t<label class=\"mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300\" for=\"appPort\">Container port override</label>",
    "advanced port conditional opening",
)
replace_once(
    "\t\t\t\t\t\t</div>\n\t\t\t\t\t</details>\n\t\t\t\t{/if}\n\t\t\t\t\n\t\t\t\t{#if (form.deployMode === 'compose' || form.deployMode === 'dockerfile')",
    "\t\t\t\t\t\t</div>\n\t\t\t\t\t\t{/if}\n\t\t\t\t\t</details>\n\t\t\t\t{/if}\n\t\t\t\t\n\t\t\t\t{#if (form.deployMode === 'compose' || form.deployMode === 'dockerfile')",
    "advanced port conditional closing",
)

replace_once(
    "\t\t\t\t{#if form.deployMode === 'compose'}\n\t\t\t\t\t<div class=\"rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40\">",
    "\t\t\t\t{#if form.deployMode === 'compose'}\n\t\t\t\t\t<details class=\"rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40\">\n\t\t\t\t\t\t<summary class=\"cursor-pointer text-sm font-medium text-gray-950 dark:text-white\">Advanced Compose settings</summary>\n\t\t\t\t\t\t<div class=\"mt-3\">",
    "compose advanced opening",
)
replace_once(
    "\t\t\t\t\t\t{/if}\n\t\t\t\t\t</div>\n\t\t\t\t{/if}\n\n\t\t\t\t\t{#if handoffPrompt}",
    "\t\t\t\t\t\t{/if}\n\t\t\t\t\t\t</div>\n\t\t\t\t\t</details>\n\t\t\t\t{/if}\n\n\t\t\t\t\t{#if error && handoffPrompt}",
    "compose advanced closing and handoff",
)

old_resources = """\t\t\t<SectionPanel
\t\t\t\ttitle=\"Resources\"
\t\t\t\tdescription=\"Keep defaults small for the self-hosted VM quota, or switch to custom values when needed.\"
\t\t\t>
\t\t\t\t<div class=\"grid gap-4 sm:grid-cols-3\">"""
new_resources = """\t\t\t<SectionPanel
\t\t\t\ttitle=\"Resources\"
\t\t\t\tdescription=\"MyPaas selects a conservative default from the detected runtime; tune it only when needed.\"
\t\t\t>
\t\t\t\t<div class=\"soft-panel mb-3 flex flex-col gap-1 p-3 text-sm sm:flex-row sm:items-center sm:justify-between\">
\t\t\t\t\t<div>
\t\t\t\t\t\t<p class=\"font-medium text-gray-950 dark:text-white\">{selectedProfile?.title ?? form.resourceProfile}</p>
\t\t\t\t\t\t<p class=\"mt-0.5 text-xs text-gray-500 dark:text-gray-400\">Recommended starting allocation.</p>
\t\t\t\t\t</div>
\t\t\t\t\t<p class=\"font-mono text-xs text-gray-700 dark:text-gray-300\">{form.memoryMb} MB · {form.cpuLimit} CPU</p>
\t\t\t\t</div>
\t\t\t\t<details class=\"rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40\">
\t\t\t\t\t<summary class=\"cursor-pointer text-sm font-medium text-gray-950 dark:text-white\">Customize resources</summary>
\t\t\t\t\t<div class=\"mt-3 grid gap-4 sm:grid-cols-3\">"""
replace_once(old_resources, new_resources, "resource summary opening")
replace_once(
    "\t\t\t\t</div>\n\t\t\t\t<p class=\"mt-3 text-xs text-gray-500 dark:text-gray-400\">Changing memory or CPU directly switches the profile to Custom.</p>\n\t\t\t</SectionPanel>\n\n\t\t\t<SectionPanel\n\t\t\t\ttitle=\"Environment\"",
    "\t\t\t\t\t</div>\n\t\t\t\t\t<p class=\"mt-3 text-xs text-gray-500 dark:text-gray-400\">Changing memory or CPU directly switches the profile to Custom.</p>\n\t\t\t\t</details>\n\t\t\t</SectionPanel>\n\n\t\t\t<SectionPanel\n\t\t\t\ttitle=\"Environment\"",
    "resource summary closing",
)

old_db = """\t\t\t\t\t\t{#if form.deployMode !== 'static'}
\t\t\t\t\t\t\t<label class=\"inline-flex min-h-8 items-center gap-2 text-sm text-gray-600 dark:text-gray-300\">
\t\t\t\t\t\t\t\t<input type=\"checkbox\" bind:checked={form.sharedPostgres} class=\"h-4 w-4 rounded border-gray-300 text-gray-950 focus:ring-gray-950 dark:border-gray-700\" />
\t\t\t\t\t\t\t\tShared PostgreSQL
\t\t\t\t\t\t\t</label>
\t\t\t\t\t\t{/if}"""
replace_once(old_db, "", "database header control")

replace_once(
    "\t\t\t\t<div>\n\t\t\t\t\t<div class=\"overflow-hidden rounded-md border border-gray-200 dark:border-gray-800\">",
    """\t\t\t\t<div>
\t\t\t\t\t{#if form.deployMode !== 'static'}
\t\t\t\t\t\t<div class=\"soft-panel mb-4 flex flex-col gap-3 p-3 text-sm sm:flex-row sm:items-center sm:justify-between\">
\t\t\t\t\t\t\t<div>
\t\t\t\t\t\t\t\t<p class=\"font-medium text-gray-950 dark:text-white\">Database</p>
\t\t\t\t\t\t\t\t<p class=\"mt-0.5 text-xs text-gray-500 dark:text-gray-400\">Optional managed PostgreSQL for projects that need it.</p>
\t\t\t\t\t\t\t</div>
\t\t\t\t\t\t\t<label class=\"inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300\">
\t\t\t\t\t\t\t\t<input type=\"checkbox\" bind:checked={form.sharedPostgres} class=\"h-4 w-4 rounded border-gray-300 text-gray-950 focus:ring-gray-950 dark:border-gray-700\" />
\t\t\t\t\t\t\t\tUse shared PostgreSQL
\t\t\t\t\t\t\t</label>
\t\t\t\t\t\t</div>
\t\t\t\t\t{/if}
\t\t\t\t\t{#if missingRequiredEnvKeys.length > 0}
\t\t\t\t\t\t<div class=\"mb-4 rounded-md border border-amber-200 bg-amber-50 px-3 py-3 text-sm text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-100\">
\t\t\t\t\t\t\t<p class=\"font-medium\">{missingRequiredEnvKeys.length} required environment value{missingRequiredEnvKeys.length === 1 ? '' : 's'} missing</p>
\t\t\t\t\t\t\t<p class=\"mt-1 font-mono text-xs\">{missingRequiredEnvKeys.join(', ')}</p>
\t\t\t\t\t\t</div>
\t\t\t\t\t{/if}
\t\t\t\t\t<div class=\"overflow-hidden rounded-md border border-gray-200 dark:border-gray-800\">""",
    "environment hierarchy",
)

replace_once(
    'description="Confirm route, runtime, and quota before create."',
    'description="Confirm the source and deployment plan before create."',
    "review description",
)

path.write_text(text)
print("new project UX patch applied")
