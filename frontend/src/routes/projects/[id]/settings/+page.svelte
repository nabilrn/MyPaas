<script lang="ts">
	import { Check, CircleAlert, Copy, Eye, EyeOff, RefreshCw, RotateCcw, Save, Trash2, X } from '@lucide/svelte';
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import ActionButton from '$components/ActionButton.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import IconButton from '$components/IconButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { ComposeResourceSummary, Project, RepoTreeEntry, ResourceProfile } from '$types';
	import { webhookURL } from '$lib/utils/urls';

	let project: Project | null = null;
	let composeResources: ComposeResourceSummary | null = null;
	let loading = true;
	let loadError = '';
	let composeResourceError = '';
	let repoInspectError = '';
	let repoInspectMessage = '';
	let repoTree: RepoTreeEntry[] = [];
	let repoTreeTruncated = false;
	let inspectingRepo = false;
	let repoInspectRequest = 0;
	let lastRepoInspectKey = '';

	let name = '';
	let branch = '';
	let imageRef = '';
	let appPort = 3000;
	let mainService = '';
	let resourceProfile: ResourceProfile = 'custom';
	let memoryMb = 512;
	let cpuLimit = 0.5;
	let composeFilePath = '';
	let composeOverridePaths = '';
	let composeProfiles = '';
	let composeWorkdir = '';
	let staticFrontendPath = '';
	let baseDirectory = '';
	let serviceResourcesStr = '{}';
	let originalServiceResourcesStr = '{}';
	let deleteInput = '';
	let showWebhookSecret = false;
	let savingSettings = false;
	let regeneratingSecret = false;
	let deletingProject = false;
	let loadingComposeResources = false;
	let resettingComposeResources = false;
	let confirmRegenerateSecret = false;
	let confirmResetComposeResources = false;
	let showWebhookHelp = false;
	let copiedTarget: 'webhook-url' | 'webhook-secret' | '' = '';
	let copiedResetTimer: ReturnType<typeof setTimeout> | undefined;

	const resourceProfiles: Array<{ id: ResourceProfile; title: string; memoryMb: number; cpuLimit: number }> = [
		{ id: 'node-python', title: 'Node/Python', memoryMb: 256, cpuLimit: 0.35 },
		{ id: 'go-small', title: 'Go small', memoryMb: 128, cpuLimit: 0.2 },
		{ id: 'compose-main', title: 'Compose main', memoryMb: 256, cpuLimit: 0.35 },
		{ id: 'static', title: 'Static/no-runtime', memoryMb: 64, cpuLimit: 0.1 },
		{ id: 'custom', title: 'Custom', memoryMb: 512, cpuLimit: 0.5 }
	];

	$: settingsChanged = project && (
		(project.sourceType === 'git' && branch !== project.branch) ||
		(project.sourceType === 'registry' && imageRef !== (project.imageRef || '')) ||
		(project.deployMode !== 'static' && appPort !== project.appPort) ||
		(project.deployMode === 'compose' && (mainService || '') !== (project.mainService || '')) ||
		(project.deployMode === 'compose' && composeFilePath !== (project.composeFilePath || '')) ||
		(project.deployMode === 'compose' && composeWorkdir !== (project.composeWorkdir || '')) ||
		(project.deployMode === 'compose' && composeOverridePaths !== (project.composeOverridePaths || []).join(', ')) ||
		(project.deployMode === 'compose' && composeProfiles !== (project.composeProfiles || []).join(', ')) ||
		staticFrontendPath !== (project.staticFrontendPath || '') ||
		baseDirectory !== (project.baseDirectory || '') ||
		serviceResourcesStr !== originalServiceResourcesStr ||
		resourceProfile !== project.resourceProfile ||
		memoryMb !== project.memoryLimitMb || cpuLimit !== project.cpuLimit
	);
	$: repoDirectorySuggestions = repoTree
		.filter((entry) => entry.type === 'directory' && entry.path !== baseDirectory.trim())
		.slice(0, 6);
	$: gitSourceChanged = project?.sourceType === 'git'
		&& (branch !== project.branch || baseDirectory !== (project.baseDirectory || ''));
	$: repoValidationStale = gitSourceChanged && repositoryInspectionKey() !== lastRepoInspectKey;
	$: publicWebhookURL = project ? webhookURL(project.id, $page.url.origin) : '';
	$: composeResourceTotal = composeResources
		? composeResources.containers + composeResources.volumes + composeResources.networks
		: 0;

	onMount(() => {
		void load();
	});

	onDestroy(() => {
		if (copiedResetTimer) clearTimeout(copiedResetTimer);
	});

	async function load() {
		loading = true;
		loadError = '';
		try {
			project = await api.projects.get($page.params.id ?? '');
			name = project.name;
			branch = project.branch;
			imageRef = project.imageRef ?? '';
			appPort = project.appPort;
			mainService = project.mainService ?? '';
			resourceProfile = project.resourceProfile;
			memoryMb = project.memoryLimitMb;
			cpuLimit = project.cpuLimit;
			composeFilePath = project.composeFilePath ?? '';
			composeOverridePaths = (project.composeOverridePaths ?? []).join(', ');
			composeProfiles = (project.composeProfiles ?? []).join(', ');
			composeWorkdir = project.composeWorkdir ?? '';
			staticFrontendPath = project.staticFrontendPath ?? '';
			baseDirectory = project.baseDirectory ?? '';
			serviceResourcesStr = JSON.stringify(project.serviceResources || {}, null, 2);
			originalServiceResourcesStr = serviceResourcesStr;
			if (project.deployMode === 'compose') await loadComposeResources(project.id);
			if (project.sourceType === 'git') void inspectRepository(false, true).catch(() => undefined);
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load project settings';
		} finally {
			loading = false;
		}
	}

	async function loadComposeResources(projectId = project?.id) {
		if (!projectId) return;
		loadingComposeResources = true;
		composeResourceError = '';
		try {
			composeResources = await api.projects.composeResources(projectId);
		} catch (err) {
			composeResourceError = err instanceof Error ? err.message : 'Failed to load Compose resources';
		} finally {
			loadingComposeResources = false;
		}
	}

	function applyResourceProfile(id: ResourceProfile) {
		const profile = resourceProfiles.find((item) => item.id === id);
		if (!profile) return;
		resourceProfile = profile.id;
		memoryMb = profile.memoryMb;
		cpuLimit = profile.cpuLimit;
	}

	function markCustomProfile() {
		resourceProfile = 'custom';
	}

	function repositoryInspectionKey() {
		if (!project || project.sourceType !== 'git') return '';
		return `${project.repoUrl}\n${branch.trim()}\n${baseDirectory.trim()}`;
	}

	function clearRepositoryValidation() {
		repoInspectError = '';
		repoInspectMessage = '';
		repoTree = [];
		repoTreeTruncated = false;
		lastRepoInspectKey = '';
	}

	function handleBranchInput(event: Event) {
		branch = (event.currentTarget as HTMLInputElement).value;
		clearRepositoryValidation();
	}

	function handleBaseDirectoryInput(event: Event) {
		baseDirectory = (event.currentTarget as HTMLInputElement).value;
		clearRepositoryValidation();
	}

	async function inspectRepository(showToast = false, force = false) {
		if (!project || project.sourceType !== 'git') return true;
		const repoUrl = project.repoUrl.trim();
		if (!repoUrl) return true;
		const requestKey = repositoryInspectionKey();
		if (!force && requestKey === lastRepoInspectKey && !repoInspectError) return true;

		const requestId = ++repoInspectRequest;
		inspectingRepo = true;
		repoInspectError = '';
		try {
			const inspection = await api.projects.inspectRepository({
				repoUrl,
				branch: branch.trim(),
				baseDirectory: baseDirectory.trim() || undefined
			});
			if (requestId !== repoInspectRequest) return false;
			if (!branch.trim() && inspection.branch) branch = inspection.branch;
			repoTree = inspection.tree ?? [];
			repoTreeTruncated = inspection.treeTruncated ?? false;
			repoInspectMessage = `Repository validated on ${inspection.branch || branch || 'default branch'}`;
			lastRepoInspectKey = repositoryInspectionKey();
			if (showToast) toast.success('Repository validated');
			return true;
		} catch (err) {
			if (requestId !== repoInspectRequest) return false;
			const message = err instanceof Error ? err.message : 'Failed to inspect repository';
			repoInspectError = message;
			repoInspectMessage = '';
			repoTree = [];
			repoTreeTruncated = false;
			lastRepoInspectKey = '';
			if (showToast) toast.error(message);
			return false;
		} finally {
			if (requestId === repoInspectRequest) inspectingRepo = false;
		}
	}

	async function validateRepositoryBeforeSave() {
		if (!project || project.sourceType !== 'git') return true;
		if (!gitSourceChanged) return true;
		if (repositoryInspectionKey() === lastRepoInspectKey && !repoInspectError) return true;
		return inspectRepository(false, true);
	}

	async function handleSave() {
		if (!project || savingSettings) return;
		savingSettings = true;
		try {
			if (!(await validateRepositoryBeforeSave())) {
				toast.error(repoInspectError || 'Repository settings could not be validated');
				savingSettings = false;
				return;
			}
			let parsedResources = {};
			try {
				parsedResources = JSON.parse(serviceResourcesStr || '{}');
			} catch {
				toast.error('Invalid JSON in service resources');
				savingSettings = false;
				return;
			}
			const payload: Record<string, unknown> = {
				...(project.sourceType === 'git' ? { branch } : { imageRef: imageRef.trim() }),
				resourceProfile,
				appPort: Number(appPort),
				memoryLimitMb: Number(memoryMb),
				cpuLimit: Number(cpuLimit),
				staticFrontendPath: project.sourceType === 'git' ? staticFrontendPath.trim() || null : null,
				baseDirectory: project.sourceType === 'git' ? baseDirectory.trim() || null : null,
				serviceResources: parsedResources
			};
			if (project.deployMode === 'compose') {
				payload.mainService = mainService.trim() || null;
				payload.composeFilePath = composeFilePath.trim() || null;
				payload.composeOverridePaths = splitCommaList(composeOverridePaths);
				payload.composeProfiles = splitCommaList(composeProfiles);
				payload.composeWorkdir = composeWorkdir.trim() || null;
			}
			project = await api.projects.update(project.id, payload);
			originalServiceResourcesStr = serviceResourcesStr;
			lastRepoInspectKey = repositoryInspectionKey();
			toast.success('Settings saved');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save settings');
		} finally {
			savingSettings = false;
		}
	}

	function splitCommaList(value: string): string[] {
		return value.split(',').map((entry) => entry.trim()).filter(Boolean);
	}

	function requestRegenerateSecret() {
		confirmRegenerateSecret = true;
	}

	function handleWindowKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && showWebhookHelp) showWebhookHelp = false;
	}

	async function handleRegenerateSecret() {
		if (!project || regeneratingSecret) return;
		regeneratingSecret = true;
		try {
			const result = await api.projects.regenerateWebhookSecret(project.id);
			project = { ...project, webhookSecret: result.webhookSecret };
			showWebhookSecret = true;
			confirmRegenerateSecret = false;
			toast.success('Webhook secret regenerated');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to regenerate webhook secret');
		} finally {
			regeneratingSecret = false;
		}
	}

	function requestResetComposeResources() {
		confirmResetComposeResources = true;
	}

	async function handleResetComposeResources() {
		if (!project || resettingComposeResources) return;
		resettingComposeResources = true;
		try {
			await api.projects.resetComposeResources(project.id);
			project = { ...project, status: 'stopped', allocatedPort: null, activeDeploymentId: null };
			await loadComposeResources(project.id);
			confirmResetComposeResources = false;
			toast.success('Compose resources reset');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to reset Compose resources');
		} finally {
			resettingComposeResources = false;
		}
	}

	function copyWebhookURL(projectId: string) {
		copyText(webhookURL(projectId, $page.url.origin), 'Webhook URL copied', 'webhook-url');
	}

	function copyText(value: string, successMessage: string, target: 'webhook-url' | 'webhook-secret') {
		navigator.clipboard?.writeText(value)
			.then(() => {
				copiedTarget = target;
				if (copiedResetTimer) clearTimeout(copiedResetTimer);
				copiedResetTimer = setTimeout(() => {
					copiedTarget = '';
					copiedResetTimer = undefined;
				}, 1800);
				toast.success(successMessage);
			})
			.catch(() => toast.error('Failed to copy'));
	}

	async function handleDelete() {
		if (!project || deleteInput !== project.name || deletingProject) return;
		deletingProject = true;
		try {
			await api.projects.delete(project.id);
			toast.success('Project deleted');
			await goto('/projects');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete project');
			deletingProject = false;
		}
	}
</script>

<svelte:window on:keydown={handleWindowKeydown} />

<svelte:head>
	<title>Settings · MyPaas</title>
</svelte:head>

{#if loading}
	<div class="space-y-4">
		<div class="surface h-44 animate-pulse"></div>
		<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_20rem]">
			<div class="surface h-60 animate-pulse"></div>
			<div class="surface h-60 animate-pulse"></div>
		</div>
	</div>
{:else if loadError || !project}
	<div class="surface overflow-hidden">
		<ErrorState title="Could not load settings" message={loadError || 'Project not found'} on:retry={() => void load()} />
	</div>
{:else if project}
	<div class="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_20rem]">
		<div class="space-y-4">
			<SectionPanel title="General" description="Source, routing, and runtime settings.">
				<div class="grid gap-4 sm:grid-cols-2">
					<div class="sm:col-span-2">
						<label class="field-label" for="pname">Project name</label>
						<input id="pname" type="text" value={name} class="field w-full bg-gray-50 text-gray-500 dark:bg-neutral-900 dark:text-gray-400" readonly />
						<p class="field-hint">Project identity and subdomain are fixed after creation.</p>
					</div>

					{#if project.sourceType === 'registry'}
						<div class="sm:col-span-2">
							<label class="field-label" for="imageRef">Container image</label>
							<input id="imageRef" type="text" bind:value={imageRef} placeholder="ghcr.io/example/app:latest" class="field w-full font-mono" />
							<p class="field-hint">Changing the public OCI image reference takes effect on the next deploy.</p>
						</div>
					{:else}
						<div>
							<label class="field-label" for="pbranch">Deploy branch</label>
							<input id="pbranch" type="text" value={branch} on:input={handleBranchInput} class="field w-full font-mono" />
						</div>
						<div>
							<label class="field-label" for="baseDirectory">Base directory</label>
							<input id="baseDirectory" type="text" value={baseDirectory} on:input={handleBaseDirectoryInput} placeholder="/" class="field w-full font-mono" />
							<p class="field-hint">Deploy from a specific subdirectory instead of the repository root.</p>
						</div>
						<div class="sm:col-span-2 border-t border-gray-100 pt-3 dark:border-neutral-800">
							<div class="flex flex-wrap items-center justify-between gap-3">
								<p class="inline-flex min-w-0 items-center gap-2 text-sm {repoInspectError ? 'text-red-700 dark:text-red-300' : repoInspectMessage && !repoValidationStale ? 'text-gray-600 dark:text-gray-300' : 'text-amber-700 dark:text-amber-200'}">
									<span class="status-dot {repoInspectError ? 'bg-red-500' : inspectingRepo ? 'animate-pulse bg-gray-400' : repoInspectMessage && !repoValidationStale ? 'bg-emerald-500' : 'bg-amber-500'}"></span>
									{inspectingRepo ? 'Validating repository source…' : repoInspectError || (repoInspectMessage && !repoValidationStale ? repoInspectMessage : 'Validate the repository before saving source changes.')}
								</p>
								<ActionButton size="xs" variant="secondary" on:click={() => void inspectRepository(true, true)} loading={inspectingRepo} loadingLabel="Validating">
									<RefreshCw slot="icon" class="h-3.5 w-3.5" />
									Validate source
								</ActionButton>
							</div>
							{#if repoDirectorySuggestions.length > 0}
								<div class="mt-2 flex flex-wrap gap-1.5">
									{#each repoDirectorySuggestions as entry}
										<button type="button" class="app-focus rounded-md border border-gray-200 bg-white px-2 py-1 font-mono text-xs text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-950 dark:border-neutral-800 dark:bg-neutral-950 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:text-white" on:click={() => { baseDirectory = entry.path; clearRepositoryValidation(); }}>
											{entry.path}
										</button>
									{/each}
								</div>
							{/if}
							{#if repoTreeTruncated}<p class="field-hint">Repository tree is truncated; enter deeper paths manually if needed.</p>{/if}
						</div>
					{/if}

					{#if project.deployMode !== 'static'}
						<div>
							<label class="field-label" for="appPort">App port</label>
							<input id="appPort" type="number" min="1" max="65535" bind:value={appPort} class="field w-full font-mono" />
						</div>
					{/if}
					{#if project.deployMode === 'compose'}
						<div>
							<label class="field-label" for="mainService">Main service</label>
							<input id="mainService" type="text" bind:value={mainService} placeholder="app" class="field w-full font-mono" />
						</div>
					{/if}
					{#if project.sourceType === 'git' && (project.deployMode === 'compose' || project.deployMode === 'dockerfile')}
						<div>
							<label class="field-label" for="staticFrontendPath">Static frontend path</label>
							<input id="staticFrontendPath" type="text" bind:value={staticFrontendPath} placeholder="frontend" class="field w-full font-mono" />
							<p class="field-hint">Build and serve this directory statically alongside the backend.</p>
						</div>
					{/if}
				</div>
			</SectionPanel>

			{#if project.deployMode === 'compose'}
				<SectionPanel title="Compose configuration" description="Compose file discovery, overrides, profiles, and working-directory control.">
					<div class="grid gap-4 sm:grid-cols-2">
						<div>
							<label class="field-label" for="composeFilePath">Compose file path</label>
							<input id="composeFilePath" type="text" bind:value={composeFilePath} placeholder="auto-detect" class="field w-full font-mono" />
							<p class="field-hint">Repo-relative, for example <span class="font-mono">infra/docker-compose.yml</span>.</p>
						</div>
						<div>
							<label class="field-label" for="composeWorkdir">Working directory override</label>
							<input id="composeWorkdir" type="text" bind:value={composeWorkdir} placeholder="auto" class="field w-full font-mono" />
							<p class="field-hint">Use only when build contexts or env files resolve from another directory.</p>
						</div>
						<div>
							<label class="field-label" for="composeOverridePaths">Override files</label>
							<input id="composeOverridePaths" type="text" bind:value={composeOverridePaths} placeholder="docker-compose.prod.yml" class="field w-full font-mono" />
							<p class="field-hint">Comma-separated repo-relative files applied before the generated override.</p>
						</div>
						<div>
							<label class="field-label" for="composeProfiles">Profiles</label>
							<input id="composeProfiles" type="text" bind:value={composeProfiles} placeholder="app, worker" class="field w-full font-mono" />
							<p class="field-hint">Comma-separated <span class="font-mono">COMPOSE_PROFILES</span> values.</p>
						</div>
					</div>
				</SectionPanel>
			{/if}

			<SectionPanel title="Resource limits" description="Default limits for the main service and optional per-service overrides." contentClass="p-0">
				<div class="grid gap-4 p-4 sm:grid-cols-3">
					<div>
						<label class="field-label" for="profile">Profile</label>
						<select id="profile" bind:value={resourceProfile} on:change={() => applyResourceProfile(resourceProfile)} class="field w-full">
							{#each resourceProfiles as profile}<option value={profile.id}>{profile.title} ({profile.memoryMb} MB / {profile.cpuLimit} CPU)</option>{/each}
						</select>
					</div>
					<div>
						<label class="field-label" for="mem">Memory</label>
						<select id="mem" bind:value={memoryMb} on:change={markCustomProfile} class="field w-full">
							{#each [64, 128, 256, 512, 1024, 2048] as m}<option value={m}>{m} MB</option>{/each}
						</select>
					</div>
					<div>
						<label class="field-label" for="cpu">CPU</label>
						<select id="cpu" bind:value={cpuLimit} on:change={markCustomProfile} class="field w-full">
							{#each [0.1, 0.2, 0.25, 0.35, 0.5, 1, 2] as c}<option value={c}>{c} core{c !== 1 ? 's' : ''}</option>{/each}
						</select>
					</div>
				</div>

				<div class="border-t border-gray-100 p-4 dark:border-neutral-800">
					<label class="field-label" for="service_resources">Other service limits (JSON)</label>
					<textarea id="service_resources" bind:value={serviceResourcesStr} rows="4" class="field w-full font-mono text-sm" placeholder='&#123;&#10;  "db": &#123;&#10;    "memoryLimitMb": 512,&#10;    "cpuLimit": 0.5&#10;  &#125;&#10;&#125;'></textarea>
					<p class="field-hint">Set memory and CPU limits for non-main services. Keys are Compose service names.</p>
					<details class="mt-2 text-sm text-gray-500 dark:text-gray-400">
						<summary class="app-focus cursor-pointer select-none font-medium text-gray-700 dark:text-gray-300">Example JSON</summary>
						<pre class="code-surface mt-2">&#123;
  "db": &#123;
    "memoryLimitMb": 256,
    "cpuLimit": 0.25
  &#125;
&#125;</pre>
					</details>
				</div>

				{#if settingsChanged}
					<div class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 bg-gray-50/50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
						<p class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><span class="status-dot bg-amber-500"></span>Unsaved project configuration changes.</p>
						<ActionButton variant="primary" on:click={handleSave} disabled={project.sourceType === 'git' && repoValidationStale && !repoInspectError} loading={savingSettings} loadingLabel="Saving">
							<Save slot="icon" class="h-4 w-4" />
							Save changes
						</ActionButton>
					</div>
				{/if}
			</SectionPanel>
		</div>

		<div class="space-y-4">
			{#if project.sourceType === 'git'}
				<SectionPanel title="Webhook" description="GitHub push deployments." contentClass="p-0">
					<svelte:fragment slot="actions">
						<IconButton label="Webhook setup instructions" variant="ghost" on:click={() => (showWebhookHelp = true)}>
							<CircleAlert class="h-4 w-4" aria-hidden="true" />
						</IconButton>
					</svelte:fragment>
					<div class="divide-y divide-gray-100 dark:divide-neutral-800">
						<div class="p-4">
							<div class="mb-1.5 flex items-center justify-between gap-2">
								<p class="field-label !mb-0">Payload URL</p>
								<IconButton label={copiedTarget === 'webhook-url' ? 'Payload URL copied' : 'Copy payload URL'} variant="ghost" on:click={() => copyWebhookURL(project?.id ?? '')}>
									{#if copiedTarget === 'webhook-url'}<Check class="h-4 w-4" aria-hidden="true" />{:else}<Copy class="h-4 w-4" aria-hidden="true" />{/if}
								</IconButton>
							</div>
							<code class="code-surface block break-all">{publicWebhookURL}</code>
						</div>
						<div class="p-4">
							<div class="mb-1.5 flex items-center justify-between gap-2">
								<p class="field-label !mb-0">Secret</p>
								<div class="flex gap-1">
									<IconButton label={showWebhookSecret ? 'Hide webhook secret' : 'Show webhook secret'} variant="ghost" on:click={() => (showWebhookSecret = !showWebhookSecret)}>
										{#if showWebhookSecret}<EyeOff class="h-4 w-4" aria-hidden="true" />{:else}<Eye class="h-4 w-4" aria-hidden="true" />{/if}
									</IconButton>
									<IconButton label={copiedTarget === 'webhook-secret' ? 'Webhook secret copied' : 'Copy webhook secret'} variant="ghost" on:click={() => copyText(project?.webhookSecret ?? '', 'Webhook secret copied', 'webhook-secret')}>
										{#if copiedTarget === 'webhook-secret'}<Check class="h-4 w-4" aria-hidden="true" />{:else}<Copy class="h-4 w-4" aria-hidden="true" />{/if}
									</IconButton>
								</div>
							</div>
							<code class="code-surface block break-all">{showWebhookSecret ? project.webhookSecret : '••••••••••••••••••••••••••••••••'}</code>
						</div>
						<div class="p-4">
							{#if confirmRegenerateSecret}
								<div class="alert-warning flex-wrap items-center justify-between">
									<p class="min-w-0 flex-1">Regenerating the secret invalidates existing GitHub webhook signatures.</p>
									<div class="flex gap-2">
										<ActionButton variant="ghost" size="xs" on:click={() => (confirmRegenerateSecret = false)} disabled={regeneratingSecret}>
											<X slot="icon" class="h-3.5 w-3.5" />
											Cancel
										</ActionButton>
										<ActionButton variant="danger" size="xs" on:click={handleRegenerateSecret} loading={regeneratingSecret} loadingLabel="Regenerating">
											<RefreshCw slot="icon" class="h-3.5 w-3.5" />
											Regenerate
										</ActionButton>
									</div>
								</div>
							{:else}
								<ActionButton variant="secondary" size="sm" on:click={requestRegenerateSecret}>
									<RefreshCw slot="icon" class="h-4 w-4" />
									Regenerate secret
								</ActionButton>
							{/if}
						</div>
					</div>
				</SectionPanel>
			{/if}

			{#if project.deployMode === 'compose'}
				<SectionPanel title="Compose resources" description="Tracked Docker resources for this project." contentClass="p-0">
					<div class="grid grid-cols-3 divide-x divide-gray-100 border-b border-gray-100 text-center dark:divide-neutral-800 dark:border-neutral-800">
						<div class="p-3"><p class="metric-value text-xl font-semibold text-gray-950 dark:text-white">{composeResources?.containers ?? 0}</p><p class="metric-label mt-1">Containers</p></div>
						<div class="p-3"><p class="metric-value text-xl font-semibold text-gray-950 dark:text-white">{composeResources?.volumes ?? 0}</p><p class="metric-label mt-1">Volumes</p></div>
						<div class="p-3"><p class="metric-value text-xl font-semibold text-gray-950 dark:text-white">{composeResources?.networks ?? 0}</p><p class="metric-label mt-1">Networks</p></div>
					</div>
					<div class="space-y-3 p-4">
						{#if composeResourceTotal > 0 && !project.activeDeploymentId}<div class="alert-warning">Compose resources exist but this project has no active deployment. Reset them before deploy if they are stale leftovers.</div>{/if}
						{#if composeResourceError}<div class="alert-danger flex-wrap items-center justify-between"><span class="min-w-0 flex-1">{composeResourceError}</span><ActionButton variant="ghost" size="xs" on:click={() => loadComposeResources()}><RefreshCw slot="icon" class="h-3.5 w-3.5" />Retry</ActionButton></div>{/if}
						<div class="flex flex-wrap gap-2">
							<ActionButton variant="secondary" size="sm" on:click={() => loadComposeResources()} loading={loadingComposeResources} loadingLabel="Checking">
								<RefreshCw slot="icon" class="h-4 w-4" />
								Check resources
							</ActionButton>
							<ActionButton variant="ghostDanger" size="sm" on:click={requestResetComposeResources} disabled={composeResourceTotal === 0 || confirmResetComposeResources}>
								<RotateCcw slot="icon" class="h-4 w-4" />
								Reset resources
							</ActionButton>
						</div>
						{#if confirmResetComposeResources}
							<div class="alert-danger flex-wrap items-center justify-between">
								<p class="min-w-0 flex-1">This removes Compose containers, volumes, networks, route, and allocated port for this project.</p>
								<div class="flex gap-2">
									<ActionButton variant="ghost" size="xs" on:click={() => (confirmResetComposeResources = false)} disabled={resettingComposeResources}><X slot="icon" class="h-3.5 w-3.5" />Cancel</ActionButton>
									<ActionButton variant="danger" size="xs" on:click={handleResetComposeResources} loading={resettingComposeResources} loadingLabel="Resetting"><RotateCcw slot="icon" class="h-3.5 w-3.5" />Reset now</ActionButton>
								</div>
							</div>
						{/if}
					</div>
				</SectionPanel>
			{/if}

			<section class="surface overflow-hidden border-red-200 dark:border-red-900/60">
				<div class="border-b border-red-100 px-4 py-3 dark:border-red-900/50">
					<h2 class="panel-title text-red-700 dark:text-red-300">Danger zone</h2>
				</div>
				<div class="space-y-3 p-4">
					<p class="text-sm text-gray-600 dark:text-gray-400">Delete this project, stop containers, remove routing, and release ports.</p>
					<label class="block">
						<span class="field-label">Type <span class="font-mono text-gray-950 dark:text-white">{project.name}</span> to confirm</span>
						<input type="text" bind:value={deleteInput} placeholder={project.name} class="field w-full border-red-300 focus:border-red-600 focus:ring-red-600 dark:border-red-900" />
					</label>
					<ActionButton variant="danger" on:click={handleDelete} disabled={deleteInput !== project.name} loading={deletingProject} loadingLabel="Deleting" full>
						<Trash2 slot="icon" class="h-4 w-4" />
						Delete project
					</ActionButton>
				</div>
			</section>
		</div>
	</div>
{/if}

{#if showWebhookHelp && project && project.sourceType === 'git'}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button type="button" class="absolute inset-0 cursor-default bg-gray-950/45" aria-label="Close webhook setup" on:click={() => (showWebhookHelp = false)}></button>
		<div class="overlay relative max-h-[90vh] w-full max-w-2xl overflow-hidden" role="dialog" aria-modal="true" aria-labelledby="webhook-help-title" tabindex="-1">
			<div class="panel-header flex items-start justify-between gap-3">
				<div class="min-w-0">
					<h2 id="webhook-help-title" class="panel-title">GitHub webhook setup</h2>
					<p class="panel-description">Configure push deploys for the selected repository.</p>
				</div>
				<IconButton label="Close webhook setup" variant="ghost" on:click={() => (showWebhookHelp = false)}><X class="h-4 w-4" aria-hidden="true" /></IconButton>
			</div>

			<div class="max-h-[calc(90vh-5rem)] space-y-4 overflow-y-auto p-4">
				<div class="grid gap-3 sm:grid-cols-[7rem_minmax(0,1fr)] sm:items-start">
					<span class="metric-label pt-2">Payload URL</span>
					<div class="flex min-w-0 items-start gap-2">
						<code class="code-surface min-w-0 flex-1 break-all">{publicWebhookURL}</code>
						<IconButton label={copiedTarget === 'webhook-url' ? 'Payload URL copied' : 'Copy payload URL'} variant="ghost" on:click={() => copyWebhookURL(project?.id ?? '')}>{#if copiedTarget === 'webhook-url'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton>
					</div>

					<span class="metric-label pt-2">Secret</span>
					<div class="flex min-w-0 items-start gap-2">
						<code class="code-surface min-w-0 flex-1 break-all">{showWebhookSecret ? project.webhookSecret : '••••••••••••••••••••••••••••••••'}</code>
						<IconButton label={copiedTarget === 'webhook-secret' ? 'Webhook secret copied' : 'Copy webhook secret'} variant="ghost" on:click={() => copyText(project?.webhookSecret ?? '', 'Webhook secret copied', 'webhook-secret')}>{#if copiedTarget === 'webhook-secret'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton>
					</div>
				</div>

				<ol class="space-y-3 text-sm text-gray-700 dark:text-gray-300">
					<li class="flex gap-3"><span class="metric-value mt-0.5 w-5 shrink-0 text-xs font-semibold text-gray-500">01</span><span>Open the GitHub repository, then go to <span class="font-medium text-gray-950 dark:text-white">Settings</span> → <span class="font-medium text-gray-950 dark:text-white">Webhooks</span>.</span></li>
					<li class="flex gap-3"><span class="metric-value mt-0.5 w-5 shrink-0 text-xs font-semibold text-gray-500">02</span><span>Choose <span class="font-medium text-gray-950 dark:text-white">Add webhook</span>, paste the payload URL, and set content type to <span class="font-mono">application/json</span>.</span></li>
					<li class="flex gap-3"><span class="metric-value mt-0.5 w-5 shrink-0 text-xs font-semibold text-gray-500">03</span><span>Paste the secret, keep <span class="font-medium text-gray-950 dark:text-white">Just the push event</span> selected, and leave the webhook active.</span></li>
					<li class="flex gap-3"><span class="metric-value mt-0.5 w-5 shrink-0 text-xs font-semibold text-gray-500">04</span><span>Save it. MyPaaS deploys only when pushes target <span class="font-mono">{project.branch}</span>.</span></li>
				</ol>

				<div class="alert-neutral">GitHub does not send commit events to MyPaaS without a webhook or GitHub App. API polling is slower, noisier, and requires extra token scope.</div>
			</div>
		</div>
	</div>
{/if}
