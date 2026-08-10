from pathlib import Path

PAGE = Path('frontend/src/routes/projects/new/+page.svelte')
VALIDATION = Path('frontend/src/lib/validation/project.ts')

page = PAGE.read_text()
validation = VALIDATION.read_text()

old_interface = '''export interface ProjectCreationReadinessInput {\n  name: string;\n  sourceType: "git" | "registry";\n  sourceReady: boolean;\n  deployMode: string;\n  appPort: string;\n  composeDisabledReason: string;\n  busy: boolean;\n}'''
new_interface = '''export interface ProjectCreationReadinessInput {\n  name: string;\n  sourceType: "git" | "registry";\n  sourceReady: boolean;\n  deployMode: string;\n  mainService?: string;\n  appPort: string;\n  composeDisabledReason: string;\n  busy: boolean;\n}'''
assert validation.count(old_interface) == 1, 'readiness interface marker changed'
validation = validation.replace(old_interface, new_interface)

old_readiness = '''  if (input.deployMode !== "static" && !input.appPort.trim()) {\n    return {\n      ready: false,\n      state: "Needs configuration",\n      reason: input.sourceType === "registry"\n        ? "Container port is required for registry images. Set it in Advanced runtime settings."\n        : "Application port could not be detected. Re-analyze the repository or set an Advanced override.",\n    };\n  }'''
new_readiness = '''  if (input.deployMode === "compose" && !input.mainService?.trim()) {\n    return {\n      ready: false,\n      state: "Needs configuration",\n      reason: "Choose the public Compose service before creating this project",\n    };\n  }\n  if (input.deployMode !== "static" && !input.appPort.trim()) {\n    return {\n      ready: false,\n      state: "Needs configuration",\n      reason: input.sourceType === "registry"\n        ? "Container port is required for registry images. Enter the container port below."\n        : "Application port could not be detected. Re-analyze the repository or enter the container port below.",\n    };\n  }'''
assert validation.count(old_readiness) == 1, 'readiness port block marker changed'
validation = validation.replace(old_readiness, new_readiness)
VALIDATION.write_text(validation)

old_import = "\timport PageHeader from '$components/PageHeader.svelte';\n\timport SectionPanel from '$components/SectionPanel.svelte';\n\timport SegmentedChoice from '$components/SegmentedChoice.svelte';"
new_import = "\timport InfoDisclosure from '$components/InfoDisclosure.svelte';\n\timport PageHeader from '$components/PageHeader.svelte';\n\timport SegmentedChoice from '$components/SegmentedChoice.svelte';"
assert page.count(old_import) == 1, 'component import marker changed'
page = page.replace(old_import, new_import)

old_ready_call = '''\t$: creationReadiness = projectCreationReadiness({\n\t\tname: form.name,\n\t\tsourceType: form.sourceType,\n\t\tsourceReady,\n\t\tdeployMode: form.deployMode,\n\t\tappPort: form.appPort,\n\t\tcomposeDisabledReason,\n\t\tbusy: submitting || detecting || inspectingRepo\n\t});'''
new_ready_call = '''\t$: creationReadiness = projectCreationReadiness({\n\t\tname: form.name,\n\t\tsourceType: form.sourceType,\n\t\tsourceReady,\n\t\tdeployMode: form.deployMode,\n\t\tmainService: form.mainService,\n\t\tappPort: form.appPort,\n\t\tcomposeDisabledReason,\n\t\tbusy: submitting || detecting || inspectingRepo\n\t});'''
assert page.count(old_ready_call) == 1, 'creation readiness call marker changed'
page = page.replace(old_ready_call, new_ready_call)

old_header = 'description="Create a routable deployment target from source code or a pre-built container image."'
assert page.count(old_header) == 1, 'page header marker changed'
page = page.replace(old_header, 'description="Deploy from a Git repository or container image."')

env_start_marker = '\t\t\t\t\t<div class="overflow-hidden rounded-md border border-gray-200 dark:border-gray-800">'
env_end_marker = '\n\t\t\t\t</div>\n\t\t\t</SectionPanel>\n\t\t</form>'
env_start = page.index(env_start_marker)
env_end = page.index(env_end_marker, env_start)
env_editor = page[env_start:env_end]

main_marker = '\n\t<div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_24rem]">'
main_start = page.index(main_marker)
prefix = page[:main_start]

new_tail = r'''
	<div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_20rem]">
		<form class="surface min-w-0" on:submit|preventDefault={handleSubmit}>
			<section class="p-5">
				<div class="mb-4">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Source</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Choose what MyPaas should deploy.</p>
				</div>

				<div class="grid gap-4">
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="name">Project name</label>
						<input id="name" type="text" value={form.name} on:input={handleNameInput} placeholder="my-app" class="field w-full" />
						<p class="mt-1 truncate font-mono text-[11px] text-gray-500 dark:text-gray-400">{previewHost}</p>
					</div>

					<SegmentedChoice
						label="Source"
						value={form.sourceType}
						options={sourceTypeOptions}
						on:change={(event) => chooseSourceType(event.detail as SourceType)}
					/>

					{#if form.sourceType === 'git'}
						<div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_13rem]">
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="repo">Repository URL</label>
								<input
									id="repo"
									type="text"
									value={form.repoUrl}
									placeholder="https://github.com/username/repo"
									class="field w-full font-mono"
									on:input={handleRepoUrlInput}
									on:blur={() => void inspectRepository(false).catch(() => undefined)}
								/>
							</div>
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="branch">Branch</label>
								<select
									id="branch"
									value={form.branch}
									class="field w-full font-mono"
									disabled={inspectingRepo || (!branchOptions.length && !form.branch)}
									on:change={handleBranchChange}
								>
									<option value="" disabled>{inspectingRepo ? 'Loading...' : 'Select branch'}</option>
									{#each branchOptions as branch}
										<option value={branch}>{branch}{branch === defaultBranch ? ' (default)' : ''}</option>
									{/each}
								</select>
							</div>
						</div>

						<div class="flex min-h-5 items-center justify-between gap-3 text-xs">
							<div class="min-w-0">
								{#if repoInspectError}
									<p class="text-red-600 dark:text-red-300">{repoInspectError}</p>
								{:else if inspectingRepo}
									<p class="text-gray-500 dark:text-gray-400">Validating repository…</p>
								{:else if repositoryInspectionCurrent}
									<p class="inline-flex items-center gap-1.5 text-gray-600 dark:text-gray-300"><Check class="h-3.5 w-3.5 text-brand-600 dark:text-brand-400" aria-hidden="true" /> Repository validated</p>
								{/if}
							</div>
							{#if form.repoUrl.trim()}
								<button type="button" class="shrink-0 text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white" on:click={() => void inspectRepository(true, true).catch(() => undefined)} disabled={inspectingRepo || detecting}>Refresh</button>
							{/if}
						</div>
					{:else}
						<div>
							<div class="mb-1 flex items-center gap-1">
								<label class="block text-xs font-medium text-gray-600 dark:text-gray-300" for="imageRef">Container image</label>
								<InfoDisclosure id="registry-image-info" label="About container images">Use a public Docker Hub, GHCR, or OCI-compatible image reference. Private registry credentials are not managed here yet.</InfoDisclosure>
							</div>
							<input id="imageRef" type="text" value={form.imageRef} on:input={handleImageRefInput} placeholder="ghcr.io/example/my-api:v1.4.0" class="field w-full font-mono" autocomplete="off" />
						</div>
					{/if}
				</div>
			</section>

			<section class="border-t border-gray-100 p-5 dark:border-gray-800" aria-live="polite">
				<div class="mb-3 flex flex-wrap items-center justify-between gap-3">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Detected setup</h2>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">MyPaas applies repository defaults automatically.</p>
					</div>
					{#if form.sourceType === 'git' && form.repoUrl.trim() && form.branch.trim()}
						<ActionButton variant="secondary" size="xs" type="button" on:click={() => void handleDetectMode().catch(() => undefined)} disabled={detecting || inspectingRepo} loading={detecting} loadingLabel="Analyzing...">Re-analyze</ActionButton>
					{/if}
				</div>

				<div class="flex items-start gap-2.5">
					{#if detecting || inspectingRepo}
						<span class="mt-1.5 h-2.5 w-2.5 shrink-0 animate-pulse rounded-full bg-yellow-500"></span>
					{:else if form.sourceType === 'git' && repoInspectError}
						<span class="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full bg-red-500"></span>
					{:else if form.deployMode !== 'auto' || form.sourceType === 'registry'}
						<Check class="mt-0.5 h-4 w-4 shrink-0 text-brand-600 dark:text-brand-400" aria-hidden="true" />
					{:else}
						<span class="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full bg-gray-400 dark:bg-gray-600"></span>
					{/if}
					<div class="min-w-0">
						<p class="text-sm font-medium text-gray-950 dark:text-white">
							{form.sourceType === 'registry'
								? (form.appPort ? `Container image · :${form.appPort}` : 'Container image · port required')
								: detecting || inspectingRepo
									? 'Analyzing repository…'
									: form.deployMode === 'compose'
										? `Docker Compose${form.mainService ? ` · ${form.mainService}` : ''}${form.appPort ? ` · :${form.appPort}` : ''}`
										: form.deployMode === 'dockerfile'
											? `Dockerfile${form.appPort ? ` · :${form.appPort}` : ''}`
											: form.deployMode === 'static'
												? 'Static site · served by Caddy'
												: detectionStateLabel}
						</p>
						{#if form.deployMode !== 'auto' && form.deployMode !== 'static' && form.appPort}
							<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{portStateLabel}</p>
						{:else if form.deployMode === 'auto' && !detecting && !inspectingRepo && !repoInspectError}
							<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{detectionStateBody}</p>
						{/if}
					</div>
				</div>

				{#if form.deployMode === 'compose' && !form.mainService}
					<div class="mt-4 max-w-md">
						<label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-200" for="mainService">Public service</label>
						{#if detectedServices.length > 0}
							<select id="mainService" bind:value={form.mainService} class="field w-full font-mono">
								<option value="">Select service</option>
								{#each detectedServices as service}<option value={service}>{service}</option>{/each}
							</select>
						{:else}
							<input id="mainService" type="text" bind:value={form.mainService} placeholder="api" class="field w-full font-mono" />
						{/if}
						<p class="mt-1 text-xs text-red-600 dark:text-red-300">Choose the service that receives public traffic.</p>
					</div>
				{/if}

				{#if form.deployMode !== 'auto' && form.deployMode !== 'static' && !form.appPort}
					<div class="mt-4 max-w-sm">
						<div class="mb-1 flex items-center gap-1">
							<label class="block text-xs font-medium text-gray-700 dark:text-gray-200" for="appPort">Container port</label>
							<InfoDisclosure id="container-port-info" label="About container ports">This is the port your app listens on inside the container. MyPaas allocates the host port and Caddy route automatically.</InfoDisclosure>
						</div>
						<input id="appPort" type="number" min="1" max="65535" value={form.appPort} placeholder="3000" on:input={handleAppPortInput} class="field w-full font-mono" />
						<p class="mt-1 text-xs text-red-600 dark:text-red-300">A container port is required before creation.</p>
					</div>
				{/if}

				{#if form.deployMode === 'compose' && composePlan}
					<div class="mt-3 text-xs">
						{#if composeBlockingIssues.length === 0}
							<p class="inline-flex items-center gap-1.5 text-gray-600 dark:text-gray-300"><Check class="h-3.5 w-3.5 text-brand-600 dark:text-brand-400" aria-hidden="true" /> Compose ready</p>
						{:else}
							<div class="border-l-2 border-red-500 pl-3 text-red-700 dark:text-red-200">
								<p class="font-medium">Compose needs attention</p>
								<p class="mt-0.5">{composeBlockingIssues[0].message}</p>
							</div>
					{/if}
					</div>
				{/if}
			</section>

			<section class="border-t border-gray-100 p-5 dark:border-gray-800">
				<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Environment</h2>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Add values the application needs at runtime.</p>
					</div>
					<div>
						<input bind:this={envFileInput} type="file" accept=".env,text/plain" class="hidden" on:change={handleEnvFileImport} />
						<ActionButton type="button" variant="secondary" size="xs" on:click={triggerEnvFileImport}>
							<span class="inline-flex items-center gap-1.5"><Upload class="h-3.5 w-3.5" aria-hidden="true" /> Import .env</span>
						</ActionButton>
					</div>
				</div>

				{#if form.deployMode !== 'static'}
					<div class="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 pb-4 dark:border-gray-800">
						<div class="flex items-center gap-1">
							<span class="text-sm text-gray-700 dark:text-gray-300">Shared PostgreSQL</span>
							<InfoDisclosure id="shared-postgres-info" label="About shared PostgreSQL">Creates a managed PostgreSQL database for this project and injects its connection URL as <span class="font-mono">DATABASE_URL</span>.</InfoDisclosure>
						</div>
						<label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
							<input type="checkbox" bind:checked={form.sharedPostgres} class="h-4 w-4 rounded border-gray-300 text-gray-950 focus:ring-gray-950 dark:border-gray-700" />
							Enable
						</label>
					</div>
				{/if}

				{#if missingRequiredEnvKeys.length > 0}
					<div class="mb-4 border-l-2 border-amber-400 pl-3 text-sm text-amber-900 dark:text-amber-100">
						<p class="font-medium">{missingRequiredEnvKeys.length} required value{missingRequiredEnvKeys.length === 1 ? '' : 's'} missing</p>
						<p class="mt-0.5 font-mono text-xs">{missingRequiredEnvKeys.join(', ')}</p>
					</div>
				{/if}

%%ENV_EDITOR%%
			</section>

			<section class="border-t border-gray-100 dark:border-gray-800">
				<details>
					<summary class="app-focus flex cursor-pointer list-none items-center justify-between px-5 py-4 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-gray-900/50 [&::-webkit-details-marker]:hidden">
						<span>Advanced</span>
						<span class="text-xs font-normal text-gray-500 dark:text-gray-400">Overrides & diagnostics</span>
					</summary>
					<div class="space-y-7 border-t border-gray-100 px-5 py-5 dark:border-gray-800">
						{#if form.sourceType === 'git'}
							<div>
								<div class="mb-3 flex items-center gap-1">
									<h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Source</h3>
									<InfoDisclosure id="project-directory-info" label="About project directory">Only set this for a monorepo. Leave it blank to deploy from the repository root.</InfoDisclosure>
								</div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="baseDirectory">Project directory</label>
								<input id="baseDirectory" type="text" value={form.baseDirectory} placeholder="Repository root" class="field w-full font-mono" on:input={handleBaseDirectoryInput} on:blur={() => void inspectRepository(false).catch(() => undefined)} />
							</div>
						{/if}

						<div>
							<div class="mb-3 flex items-center gap-1">
								<h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Runtime</h3>
								<InfoDisclosure id="runtime-override-info" label="About runtime overrides">Use these only when automatic detection is wrong or your image does not expose enough metadata.</InfoDisclosure>
							</div>
							<div class="grid gap-4">
								{#if form.sourceType === 'git'}
									<SegmentedChoice label="Deployment mode override" value={form.deployMode} options={deployModeOptions} on:change={(event) => chooseDeployMode(event.detail as DeployModeChoice)} />
								{/if}
								{#if form.deployMode === 'compose' && form.mainService}
									<div class="max-w-md">
										<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="mainServiceAdvanced">Public service override</label>
										<input id="mainServiceAdvanced" type="text" bind:value={form.mainService} class="field w-full font-mono" />
									</div>
								{/if}
								{#if form.deployMode !== 'static' && form.appPort}
									<div class="max-w-sm">
										<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="appPortAdvanced">Container port override</label>
										<input id="appPortAdvanced" type="number" min="1" max="65535" value={form.appPort} on:input={handleAppPortInput} class="field w-full font-mono" />
									</div>
								{/if}
								{#if (form.deployMode === 'compose' || form.deployMode === 'dockerfile') && staticFrontendCandidates.length > 0}
									<div class="max-w-md">
										<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="staticFrontendPath">Static frontend override</label>
										<select id="staticFrontendPath" bind:value={form.staticFrontendPath} class="field w-full">
											<option value="">Disabled</option>
											{#each staticFrontendCandidates as candidate}<option value={candidate}>{candidate}</option>{/each}
										</select>
									</div>
								{/if}
							</div>
						</div>

						{#if form.deployMode === 'compose'}
							<div>
								<div class="mb-3 flex flex-wrap items-center justify-between gap-2">
									<div class="flex items-center gap-1">
										<h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Compose</h3>
										<InfoDisclosure id="compose-overrides-info" label="About Compose overrides">Override the detected Compose file, working directory, profiles, or additional override files only when repository defaults are not enough.</InfoDisclosure>
									</div>
									<ActionButton variant="secondary" size="xs" type="button" disabled={composeCandidatesLoading || !form.repoUrl.trim() || !form.branch.trim()} loading={composeCandidatesLoading} loadingLabel="Scanning..." on:click={() => void refreshComposeCandidates(true)}>Scan files</ActionButton>
								</div>
								<div class="grid gap-4 sm:grid-cols-2">
									<div>
										<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeFilePath">Compose file</label>
										<input id="composeFilePath" type="text" bind:value={form.composeFilePath} list="compose-candidates" placeholder="Auto-detect" class="field w-full font-mono" />
										<datalist id="compose-candidates">{#each composeCandidates as candidate}<option value={candidate.path}></option>{/each}</datalist>
									</div>
									<div>
										<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeWorkdir">Working directory</label>
										<input id="composeWorkdir" type="text" bind:value={form.composeWorkdir} placeholder="Auto" class="field w-full font-mono" />
									</div>
									<div>
										<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeOverridePaths">Override files</label>
										<input id="composeOverridePaths" type="text" bind:value={form.composeOverridePaths} placeholder="docker-compose.prod.yml" class="field w-full font-mono" />
									</div>
									<div>
										<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeProfiles">Profiles</label>
										<input id="composeProfiles" type="text" bind:value={form.composeProfiles} placeholder="production" class="field w-full font-mono" />
									</div>
								</div>
								{#if composeCandidatesError}<p class="mt-2 text-xs text-red-600 dark:text-red-300">{composeCandidatesError}</p>{/if}
							</div>
						{/if}

						<div>
							<div class="mb-3 flex items-center gap-1">
								<h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Resources</h3>
								<InfoDisclosure id="resource-limits-info" label="About resource limits">MyPaas selects a conservative starting profile from the detected runtime. Change it only when the workload needs different limits.</InfoDisclosure>
							</div>
							<div class="grid gap-3 sm:grid-cols-3">
								<div>
									<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="profile">Profile</label>
									<select id="profile" bind:value={form.resourceProfile} on:change={() => applyResourceProfile(form.resourceProfile)} class="field w-full">
										{#each resourceProfiles as profile}<option value={profile.id}>{profile.title}</option>{/each}
									</select>
								</div>
								<div>
									<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="memory">Memory</label>
									<select id="memory" bind:value={form.memoryMb} on:change={markCustomProfile} class="field w-full">
										<option value="64">64 MB</option><option value="128">128 MB</option><option value="256">256 MB</option><option value="512">512 MB</option><option value="1024">1024 MB</option><option value="2048">2048 MB</option>
									</select>
								</div>
								<div>
									<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="cpu">CPU</label>
									<select id="cpu" bind:value={form.cpuLimit} on:change={markCustomProfile} class="field w-full">
										<option value="0.1">0.10</option><option value="0.2">0.20</option><option value="0.25">0.25</option><option value="0.35">0.35</option><option value="0.5">0.50</option><option value="1">1.00</option><option value="2">2.00</option>
									</select>
								</div>
							</div>
						</div>

						<div>
							<h3 class="mb-3 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Diagnostics</h3>
							{#if form.sourceType === 'git'}
								<div class="mb-5">
									<div class="mb-2 flex items-center justify-between gap-2">
										<p class="text-xs font-medium text-gray-700 dark:text-gray-200">Repository structure</p>
										{#if repoTreeTruncated}<span class="text-[11px] text-gray-500 dark:text-gray-400">First {repoTree.length} entries</span>{/if}
									</div>
									<div class="max-h-60 overflow-auto border-y border-gray-100 text-xs dark:border-gray-800">
										{#if repoTree.length > 0}
											{#each repoTree as item}
												<div class="flex items-center gap-2 border-b border-gray-100 px-1 py-1.5 last:border-b-0 dark:border-gray-800" style={`padding-left: ${0.25 + item.depth * 0.8}rem;`}>
													<span class="w-7 shrink-0 text-[10px] uppercase text-gray-400">{item.type === 'directory' ? 'dir' : 'file'}</span>
													<span class="truncate font-mono text-gray-600 dark:text-gray-300">{item.path}</span>
												</div>
											{/each}
										{:else}
											<p class="py-3 text-gray-500 dark:text-gray-400">Repository structure is available after validation.</p>
										{/if}
									</div>
								</div>
							{/if}

							{#if form.deployMode === 'compose' && composePlan}
								<div class="space-y-3">
									<div class="grid gap-2 text-xs sm:grid-cols-2">
										<p><span class="text-gray-500 dark:text-gray-400">Recommended public service</span><br /><span class="font-mono text-gray-950 dark:text-white">{composePlan.recommendedMainService}:{composePlan.recommendedAppPort}</span></p>
										<p><span class="text-gray-500 dark:text-gray-400">Required env</span><br /><span class="font-mono text-gray-950 dark:text-white">{composePlan.requiredEnvVars.length > 0 ? composePlan.requiredEnvVars.join(', ') : '-'}</span></p>
									</div>
									<div class="divide-y divide-gray-100 border-y border-gray-100 dark:divide-gray-800 dark:border-gray-800">
										{#each composePlan.services as service}
											<div class="grid gap-1 py-2 text-xs sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
												<div class="min-w-0"><span class="font-mono font-medium text-gray-950 dark:text-white">{service.name}</span><span class="ml-2 text-gray-500 dark:text-gray-400">{service.buildContext ? `build ${service.buildContext}` : service.image ? service.image : 'no build/image'}</span></div>
												<span class="font-mono text-gray-500 dark:text-gray-400">{service.role} · {formatComposeServicePorts(service)}</span>
											</div>
										{/each}
									</div>
									{#if composePlan.issues.length > 0}
										<div class="space-y-2">
											{#each composePlan.issues as issue}
												<div class="border-l-2 pl-3 text-xs {issue.severity === 'error' ? 'border-red-500 text-red-700 dark:text-red-200' : issue.severity === 'warning' ? 'border-yellow-500 text-yellow-800 dark:text-yellow-100' : 'border-gray-300 text-gray-600 dark:border-gray-700 dark:text-gray-300'}">
													<p class="font-medium">{issueLabel(issue)}</p><p class="mt-0.5">{issue.message}</p>
												</div>
											{/each}
										</div>
									{/if}
								</div>
							{/if}

							{#if error && handoffPrompt}
								<div class="mt-5 border-l-2 border-gray-300 pl-3 dark:border-gray-700">
									<div class="flex flex-wrap items-center justify-between gap-2">
										<div><p class="text-xs font-medium text-gray-700 dark:text-gray-200">Coding-agent handoff</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Use only when repository changes are required to satisfy deployability.</p></div>
										<ActionButton variant="secondary" size="xs" type="button" on:click={copyHandoffPrompt}>{copiedHandoffPrompt === handoffPrompt ? 'Copied' : 'Copy prompt'}</ActionButton>
									</div>
								</div>
							{/if}
						</div>
					</div>
				</details>
			</section>

			<div class="border-t border-gray-100 p-5 lg:hidden dark:border-gray-800">
				<ActionButton variant="primary" size="md" type="submit" full loading={submitting} loadingLabel="Creating..." disabled={!canSubmit}>Create project</ActionButton>
				{#if createDisabledReason}<p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{createDisabledReason}</p>{/if}
			</div>
		</form>

		<aside class="hidden lg:block lg:sticky lg:top-6 lg:self-start">
			<div class="surface overflow-hidden">
				<div class="panel-header">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Review</h2>
				</div>
				<dl class="divide-y divide-gray-100 text-sm dark:divide-gray-800">
					<div class="px-4 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">Hostname</dt><dd class="mt-1 truncate font-mono font-medium text-gray-950 dark:text-white">{previewHost}</dd></div>
					<div class="px-4 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Source</dt>
						<dd class="mt-1 min-w-0 text-gray-950 dark:text-white">
							<p class="truncate font-mono">{form.sourceType === 'registry' ? (form.imageRef || '-') : (form.repoUrl || '-')}</p>
							{#if form.sourceType === 'git'}<p class="mt-0.5 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{form.branch || '-'}</p>{/if}
						</dd>
					</div>
					<div class="px-4 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Runtime</dt>
						<dd class="mt-1 text-gray-950 dark:text-white">{form.sourceType === 'registry' ? 'Container image' : form.deployMode === 'compose' ? 'Docker Compose' : form.deployMode === 'dockerfile' ? 'Dockerfile' : form.deployMode === 'static' ? 'Static site' : 'Analyzing'}</dd>
					</div>
					<div class="px-4 py-3" aria-live="polite">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Status</dt>
						<dd class="mt-1 font-medium {canSubmit ? 'text-brand-700 dark:text-brand-300' : 'text-gray-700 dark:text-gray-200'}">{reviewStateLabel}</dd>
						{#if createDisabledReason}<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{createDisabledReason}</p>{/if}
					</div>
				</dl>
				<div class="border-t border-gray-100 p-4 dark:border-gray-800">
					<ActionButton variant="primary" size="md" type="button" full on:click={handleSubmit} loading={submitting} loadingLabel="Creating..." disabled={!canSubmit}>Create project</ActionButton>
				</div>
			</div>
		</aside>
	</div>
</div>'''

new_tail = new_tail.replace('%%ENV_EDITOR%%', env_editor)
PAGE.write_text(prefix + '\n' + new_tail.lstrip('\n'))

print('patched new project visual hierarchy')
