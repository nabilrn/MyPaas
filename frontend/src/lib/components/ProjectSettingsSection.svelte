<script lang="ts">
	import { Check, Copy, Eye, EyeOff, RefreshCw, Trash2, X } from '@lucide/svelte';
	import { onDestroy, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import IconButton from '$components/IconButton.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';
	import ProjectEffectiveConfiguration from '$components/ProjectEffectiveConfiguration.svelte';
	import SelectMenu from '$components/SelectMenu.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { ComposeResourceSummary, Project, RepoTreeEntry, ResourceProfile } from '$types';
	import { projectURL, webhookURL } from '$lib/utils/urls';

	export let section: 'general' | 'source' | 'resources' | 'webhook' | 'danger' = 'general';

	let project: Project | null = null;
	let composeResources: ComposeResourceSummary | null = null;
	let loading = true;
	let loadError = '';
	let composeResourceError = '';
	let repoInspectError = '';
	let repoTree: RepoTreeEntry[] = [];
	let repoBranches: string[] = [];
	let inspectingRepo = false;
	let repoInspectRequest = 0;
	let lastRepoInspectKey = '';
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
	let confirmRegenerateSecret = false;
	let showWebhookHelp = false;
	let copiedTarget: 'webhook-url' | 'webhook-secret' | '' = '';
	let copiedResetTimer: ReturnType<typeof setTimeout> | undefined;

	let resourceProfiles: Array<{ id: ResourceProfile; title: string; memoryMb: number; cpuLimit: number }> = [
		{ id: 'node-python', title: 'Node/Python', memoryMb: 256, cpuLimit: 0.35 },
		{ id: 'go-small', title: 'Go small', memoryMb: 128, cpuLimit: 0.2 },
		{ id: 'compose-main', title: 'Compose main', memoryMb: 256, cpuLimit: 0.35 },
		{ id: 'static', title: 'Static/no-runtime', memoryMb: 64, cpuLimit: 0.1 },
		{ id: 'custom', title: 'Custom', memoryMb: 512, cpuLimit: 0.5 }
	];

	$: sourceChanged = Boolean(project && (
		(project.sourceType === 'git' && branch !== project.branch) ||
		(project.sourceType === 'registry' && imageRef !== (project.imageRef || '')) ||
		(project.deployMode !== 'static' && appPort !== project.appPort) ||
		(project.deployMode === 'compose' && (mainService || '') !== (project.mainService || '')) ||
		(project.deployMode === 'compose' && composeFilePath !== (project.composeFilePath || '')) ||
		(project.deployMode === 'compose' && composeWorkdir !== (project.composeWorkdir || '')) ||
		(project.deployMode === 'compose' && composeOverridePaths !== (project.composeOverridePaths || []).join(', ')) ||
		(project.deployMode === 'compose' && composeProfiles !== (project.composeProfiles || []).join(', ')) ||
		staticFrontendPath !== (project.staticFrontendPath || '') ||
		baseDirectory !== (project.baseDirectory || '')
	));
	$: resourcesChanged = Boolean(project && (
		serviceResourcesStr !== originalServiceResourcesStr ||
		resourceProfile !== project.resourceProfile ||
		memoryMb !== project.memoryLimitMb ||
		cpuLimit !== project.cpuLimit
	));
	$: gitSourceChanged = Boolean(project?.sourceType === 'git' && (branch !== project.branch || baseDirectory !== (project.baseDirectory || '')));
	$: publicWebhookURL = project ? webhookURL(project.id, $page.url.origin) : '';
	$: effectivePublicURL = project ? projectURL(project.subdomain, $page.url.protocol, $page.url.hostname) : '';
	$: branchOptions = Array.from(new Set([branch, ...repoBranches].filter(Boolean))).map((item) => ({ value: item, label: item }));
	$: baseDirectoryOptions = [
		{ value: '', label: 'Repository root' },
		...Array.from(new Set([baseDirectory, ...repoTree.filter((entry) => entry.type === 'directory').map((entry) => entry.path)].filter(Boolean)))
			.map((path) => ({ value: path, label: path }))
	];
	$: resourceProfileOptions = resourceProfiles.map((profile) => ({
		value: profile.id,
		label: profile.title,
		description: `${profile.memoryMb} MB · ${formatCpu(profile.cpuLimit)} CPU`
	}));
	$: pageTitle = section === 'general'
		? 'General'
		: section === 'source'
			? 'Source'
			: section === 'resources'
				? 'Resources'
				: section === 'webhook'
					? 'Webhook'
					: 'Danger zone';

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

			if (section === 'resources') {
				const platformSettings = await api.admin.getSettings().catch(() => null);
				if (platformSettings) {
					const configured: Partial<Record<ResourceProfile, { memoryMb: number; cpuLimit: number }>> = {
						static: { memoryMb: platformSettings.profile_static_memory_mb ?? 64, cpuLimit: platformSettings.profile_static_cpu_limit ?? 0.1 },
						'go-small': { memoryMb: platformSettings.profile_go_small_memory_mb ?? 128, cpuLimit: platformSettings.profile_go_small_cpu_limit ?? 0.2 },
						'node-python': { memoryMb: platformSettings.profile_node_python_memory_mb ?? 256, cpuLimit: platformSettings.profile_node_python_cpu_limit ?? 0.35 },
						'compose-main': { memoryMb: platformSettings.profile_compose_main_memory_mb ?? 256, cpuLimit: platformSettings.profile_compose_main_cpu_limit ?? 0.35 }
					};
					resourceProfiles = resourceProfiles.map((profile) => ({ ...profile, ...(configured[profile.id] ?? {}) }));
				}
				if (project.deployMode === 'compose') void loadComposeResources(project.id);
			}

			if (section === 'source' && project.sourceType === 'git') void inspectRepository(false, true);
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
			composeResourceError = err instanceof Error ? err.message : 'Failed to load runtime resources';
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

	function formatCpu(value: number) {
		return Number(value.toFixed(2)).toString();
	}

	function repositoryInspectionKey() {
		if (!project || project.sourceType !== 'git') return '';
		return `${project.repoUrl}\n${branch.trim()}\n${baseDirectory.trim()}`;
	}

	function invalidateRepositoryValidation() {
		repoInspectError = '';
		lastRepoInspectKey = '';
	}

	function handleBranchChange(nextBranch: string) {
		branch = nextBranch;
		baseDirectory = '';
		invalidateRepositoryValidation();
		void inspectRepository(false, true);
	}

	function handleBaseDirectoryChange(nextDirectory: string) {
		baseDirectory = nextDirectory;
		invalidateRepositoryValidation();
		void inspectRepository(false, true);
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
			repoBranches = inspection.branches ?? [];
			repoTree = inspection.tree ?? [];
			lastRepoInspectKey = repositoryInspectionKey();
			if (showToast) toast.success('Source checked');
			return true;
		} catch (err) {
			if (requestId !== repoInspectRequest) return false;
			repoInspectError = err instanceof Error ? err.message : 'Could not check repository';
			lastRepoInspectKey = '';
			if (showToast) toast.error(repoInspectError);
			return false;
		} finally {
			if (requestId === repoInspectRequest) inspectingRepo = false;
		}
	}

	async function validateRepositoryBeforeSave() {
		if (!project || project.sourceType !== 'git' || !gitSourceChanged) return true;
		if (repositoryInspectionKey() === lastRepoInspectKey && !repoInspectError) return true;
		return inspectRepository(false, true);
	}

	async function handleSourceSave() {
		if (!project || savingSettings) return;
		savingSettings = true;
		try {
			if (!(await validateRepositoryBeforeSave())) {
				toast.error(repoInspectError || 'Source could not be checked');
				return;
			}
			const payload: Record<string, unknown> = {
				...(project.sourceType === 'git' ? { branch } : { imageRef: imageRef.trim() }),
				appPort: Number(appPort),
				staticFrontendPath: project.sourceType === 'git' ? staticFrontendPath.trim() || null : null,
				baseDirectory: project.sourceType === 'git' ? baseDirectory.trim() || null : null
			};
			if (project.deployMode === 'compose') {
				payload.mainService = mainService.trim() || null;
				payload.composeFilePath = composeFilePath.trim() || null;
				payload.composeOverridePaths = splitCommaList(composeOverridePaths);
				payload.composeProfiles = splitCommaList(composeProfiles);
				payload.composeWorkdir = composeWorkdir.trim() || null;
			}
			project = await api.projects.update(project.id, payload);
			lastRepoInspectKey = repositoryInspectionKey();
			toast.success('Source settings saved');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save source settings');
		} finally {
			savingSettings = false;
		}
	}

	async function handleResourcesSave() {
		if (!project || savingSettings) return;
		savingSettings = true;
		try {
			let parsedResources = {};
			try {
				parsedResources = JSON.parse(serviceResourcesStr || '{}');
			} catch {
				toast.error('Other service limits must be valid JSON');
				return;
			}
			project = await api.projects.update(project.id, {
				resourceProfile,
				memoryLimitMb: Number(memoryMb),
				cpuLimit: Number(cpuLimit),
				serviceResources: parsedResources
			});
			originalServiceResourcesStr = serviceResourcesStr;
			toast.success('Resource settings saved');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save resource settings');
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
<svelte:head><title>{pageTitle} · MyPaas</title></svelte:head>

{#if loading}
	<div class="surface flex min-h-64 items-center justify-center"><LoadingIndicator label={`Loading ${pageTitle.toLowerCase()}`} /></div>
{:else if loadError || !project}
	<div class="surface overflow-hidden"><ErrorState title="Could not load settings" message={loadError || 'Project not found'} on:retry={() => void load()} /></div>
{:else if project}
	{#if section === 'general'}
		<div class="w-full space-y-4">
			<div><h1 class="text-lg font-semibold text-gray-950 dark:text-white">General information</h1><p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Basic information about this project.</p></div>
			<ProjectEffectiveConfiguration {project} publicUrl={effectivePublicURL} />
		</div>

	{:else if section === 'source'}
		<div class="w-full space-y-5">
			<div><h1 class="text-lg font-semibold text-gray-950 dark:text-white">Source</h1><p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Choose what MyPaaS deploys.</p></div>

			{#if project.sourceType === 'registry'}
				<div class="border-t border-gray-100 pt-4 dark:border-neutral-800">
					<label class="field-label" for="imageRef">Container image</label>
					<input id="imageRef" type="text" bind:value={imageRef} placeholder="ghcr.io/example/app:latest" class="field w-full font-mono" />
				</div>
			{:else}
				<div class="divide-y divide-gray-100 border-y border-gray-100 dark:divide-neutral-800 dark:border-neutral-800">
					<div class="grid gap-2 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-center"><p class="text-sm text-gray-500 dark:text-gray-400">Repository</p><p class="truncate font-mono text-sm text-gray-950 dark:text-white" title={project.repoUrl}>{project.repoUrl}</p></div>
					<div class="grid gap-2 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-center"><span class="text-sm text-gray-500 dark:text-gray-400">Branch</span><SelectMenu value={branch} options={branchOptions} ariaLabel="Deployment branch" disabled={inspectingRepo || branchOptions.length === 0} on:change={(event) => handleBranchChange(event.detail)} /></div>
					<div class="grid gap-2 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-center"><span class="text-sm text-gray-500 dark:text-gray-400">Base directory</span><SelectMenu value={baseDirectory} options={baseDirectoryOptions} ariaLabel="Base directory" disabled={inspectingRepo} on:change={(event) => handleBaseDirectoryChange(event.detail)} /></div>
				</div>
				{#if repoInspectError}<div class="alert-danger flex-wrap items-center justify-between gap-3"><span class="min-w-0 flex-1">{repoInspectError}</span><ActionButton variant="ghost" size="xs" on:click={() => void inspectRepository(true, true)} loading={inspectingRepo} loadingLabel="Checking"><RefreshCw slot="icon" class="h-3.5 w-3.5" />Retry</ActionButton></div>{/if}
			{/if}

			<details class="border-y border-gray-100 py-3 dark:border-neutral-800">
				<summary class="app-focus cursor-pointer select-none text-sm font-medium text-gray-700 dark:text-gray-300">Advanced source settings</summary>
				<div class="mt-4 grid gap-4 sm:grid-cols-2">
					{#if project.deployMode !== 'static'}<div><label class="field-label" for="appPort">App port</label><input id="appPort" type="number" min="1" max="65535" bind:value={appPort} class="field w-full font-mono" /></div>{/if}
					{#if project.deployMode === 'compose'}
						<div><label class="field-label" for="mainService">Main service</label><input id="mainService" type="text" bind:value={mainService} placeholder="app" class="field w-full font-mono" /></div>
						<div><label class="field-label" for="composeFilePath">Compose file</label><input id="composeFilePath" type="text" bind:value={composeFilePath} placeholder="docker-compose.yml" class="field w-full font-mono" /></div>
						<div><label class="field-label" for="composeWorkdir">Working directory</label><input id="composeWorkdir" type="text" bind:value={composeWorkdir} placeholder="auto" class="field w-full font-mono" /></div>
						<div><label class="field-label" for="composeOverridePaths">Override files</label><input id="composeOverridePaths" type="text" bind:value={composeOverridePaths} placeholder="docker-compose.prod.yml" class="field w-full font-mono" /></div>
						<div><label class="field-label" for="composeProfiles">Profiles</label><input id="composeProfiles" type="text" bind:value={composeProfiles} placeholder="app, worker" class="field w-full font-mono" /></div>
					{/if}
					{#if project.sourceType === 'git' && (project.deployMode === 'compose' || project.deployMode === 'dockerfile')}<div><label class="field-label" for="staticFrontendPath">Static frontend path</label><input id="staticFrontendPath" type="text" bind:value={staticFrontendPath} placeholder="frontend" class="field w-full font-mono" /></div>{/if}
				</div>
			</details>
			<div class="flex flex-wrap items-center justify-between gap-3"><p class="text-sm text-gray-500 dark:text-gray-400">Changes take effect on the next deployment.</p>{#if sourceChanged}<ActionButton variant="primary" on:click={handleSourceSave} loading={savingSettings} loadingLabel="Saving">Save changes</ActionButton>{/if}</div>
		</div>

	{:else if section === 'resources'}
		<div class="w-full space-y-5">
			<div><h1 class="text-lg font-semibold text-gray-950 dark:text-white">Resources</h1><p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Set how much CPU and memory this project can use.</p></div>
			<div class="border-y border-gray-100 py-4 dark:border-neutral-800">
				<label class="field-label">Resource profile</label>
				<SelectMenu value={resourceProfile} options={resourceProfileOptions} ariaLabel="Resource profile" on:change={(event) => applyResourceProfile(event.detail as ResourceProfile)} />
				{#if resourceProfile === 'custom'}
					<div class="mt-4 grid gap-4 sm:grid-cols-2"><div><label class="field-label" for="mem">Memory (MB)</label><input id="mem" type="number" min="64" max="32768" step="1" bind:value={memoryMb} on:input={markCustomProfile} class="field w-full" /></div><div><label class="field-label" for="cpu">CPU</label><input id="cpu" type="number" min="0.1" max="32" step="0.05" bind:value={cpuLimit} on:input={markCustomProfile} class="field w-full" /></div></div>
				{:else}
					<div class="mt-4 grid gap-4 sm:grid-cols-2"><div><p class="text-xs text-gray-500 dark:text-gray-400">Memory</p><p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">{memoryMb} MB</p></div><div><p class="text-xs text-gray-500 dark:text-gray-400">CPU</p><p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">{formatCpu(cpuLimit)} CPU</p></div></div>
				{/if}
			</div>

			{#if project.deployMode === 'compose'}
				<details class="border-b border-gray-100 pb-4 dark:border-neutral-800"><summary class="app-focus cursor-pointer select-none text-sm font-medium text-gray-700 dark:text-gray-300">Advanced resource limits</summary><div class="mt-4"><label class="field-label" for="service_resources">Other services (JSON)</label><textarea id="service_resources" bind:value={serviceResourcesStr} rows="5" class="field w-full font-mono text-sm"></textarea></div></details>
				<div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 pb-4 dark:border-neutral-800"><div><p class="text-sm font-medium text-gray-950 dark:text-white">Runtime resources</p><p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{composeResources?.containers ?? 0} container{(composeResources?.containers ?? 0) === 1 ? '' : 's'} · {composeResources?.volumes ?? 0} volume{(composeResources?.volumes ?? 0) === 1 ? '' : 's'} · {composeResources?.networks ?? 0} network{(composeResources?.networks ?? 0) === 1 ? '' : 's'}</p></div><IconButton label="Refresh runtime resources" variant="secondary" loading={loadingComposeResources} on:click={() => void loadComposeResources()}><RefreshCw class="h-4 w-4" aria-hidden="true" /></IconButton></div>
				{#if composeResourceError}<div class="alert-danger">{composeResourceError}</div>{/if}
			{/if}
			<div class="flex flex-wrap items-center justify-between gap-3"><p class="text-sm text-gray-500 dark:text-gray-400">New limits apply after the next deployment.</p>{#if resourcesChanged}<ActionButton variant="primary" on:click={handleResourcesSave} loading={savingSettings} loadingLabel="Saving">Save changes</ActionButton>{/if}</div>
		</div>

	{:else if section === 'webhook'}
		<div class="w-full space-y-5">
			<div><h1 class="text-lg font-semibold text-gray-950 dark:text-white">Webhook</h1><p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Deploy when changes are pushed to GitHub.</p></div>
			{#if project.sourceType !== 'git'}
				<div class="alert-neutral">Webhook deployment is available only for projects using a Git repository source. <a href={`/projects/${project.id}/settings/source`} class="font-medium underline underline-offset-2">Open Source settings</a>.</div>
			{:else}
				<div class="divide-y divide-gray-100 border-y border-gray-100 dark:divide-neutral-800 dark:border-neutral-800">
					<div class="grid gap-2 py-3 sm:grid-cols-[9rem_minmax(0,1fr)_auto] sm:items-center"><p class="text-sm text-gray-500 dark:text-gray-400">Payload URL</p><p class="min-w-0 break-all font-mono text-sm text-gray-950 dark:text-white">{publicWebhookURL}</p><IconButton label={copiedTarget === 'webhook-url' ? 'Payload URL copied' : 'Copy payload URL'} variant="ghost" on:click={() => copyWebhookURL(project?.id ?? '')}>{#if copiedTarget === 'webhook-url'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton></div>
					<div class="grid gap-2 py-3 sm:grid-cols-[9rem_minmax(0,1fr)_auto] sm:items-center"><p class="text-sm text-gray-500 dark:text-gray-400">Secret</p><p class="min-w-0 break-all font-mono text-sm text-gray-950 dark:text-white">{showWebhookSecret ? project.webhookSecret : '••••••••••••••••'}</p><div class="flex items-center gap-1"><IconButton label={showWebhookSecret ? 'Hide webhook secret' : 'Reveal webhook secret'} variant="ghost" on:click={() => (showWebhookSecret = !showWebhookSecret)}>{#if showWebhookSecret}<EyeOff class="h-4 w-4" />{:else}<Eye class="h-4 w-4" />{/if}</IconButton><IconButton label={copiedTarget === 'webhook-secret' ? 'Webhook secret copied' : 'Copy webhook secret'} variant="ghost" on:click={() => copyText(project?.webhookSecret ?? '', 'Webhook secret copied', 'webhook-secret')}>{#if copiedTarget === 'webhook-secret'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton></div></div>
				</div>
				<div class="flex flex-wrap items-center gap-2"><ActionButton variant="secondary" on:click={() => (showWebhookHelp = true)}>Setup guide</ActionButton>{#if confirmRegenerateSecret}<ActionButton variant="ghost" on:click={() => (confirmRegenerateSecret = false)} disabled={regeneratingSecret}><X slot="icon" class="h-4 w-4" />Cancel</ActionButton><ActionButton variant="danger" on:click={handleRegenerateSecret} loading={regeneratingSecret} loadingLabel="Regenerating"><RefreshCw slot="icon" class="h-4 w-4" />Regenerate secret</ActionButton>{:else}<ActionButton variant="ghost" on:click={requestRegenerateSecret}>Regenerate secret</ActionButton>{/if}</div>
			{/if}
		</div>

	{:else if section === 'danger'}
		<div class="w-full space-y-5">
			<h1 class="text-lg font-semibold text-red-700 dark:text-red-300">Danger zone</h1>
			<section class="border-y border-red-200 py-4 dark:border-red-900/60"><h2 class="text-sm font-semibold text-red-700 dark:text-red-300">Delete project</h2><p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Permanently delete this project.</p><label class="mt-4 block"><span class="field-label">Type <span class="font-mono text-gray-950 dark:text-white">{project.name}</span> to confirm</span><input type="text" bind:value={deleteInput} placeholder={project.name} class="field w-full border-red-300 focus:border-red-600 focus:ring-red-600 dark:border-red-900" /></label><ActionButton className="mt-3" variant="danger" on:click={handleDelete} disabled={deleteInput !== project.name} loading={deletingProject} loadingLabel="Deleting"><Trash2 slot="icon" class="h-4 w-4" />Delete project</ActionButton></section>
		</div>
	{/if}
{/if}

{#if showWebhookHelp && project && project.sourceType === 'git' && section === 'webhook'}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button type="button" class="absolute inset-0 cursor-default bg-gray-950/45" aria-label="Close webhook setup" on:click={() => (showWebhookHelp = false)}></button>
		<div class="overlay relative max-h-[90vh] w-full max-w-2xl overflow-hidden" role="dialog" aria-modal="true" aria-labelledby="webhook-help-title" tabindex="-1">
			<div class="panel-header flex items-start justify-between gap-3"><div><h2 id="webhook-help-title" class="panel-title">GitHub webhook setup</h2><p class="panel-description">Connect GitHub pushes to this project.</p></div><IconButton label="Close webhook setup" variant="ghost" on:click={() => (showWebhookHelp = false)}><X class="h-4 w-4" /></IconButton></div>
			<div class="max-h-[calc(90vh-5rem)] space-y-4 overflow-y-auto p-4"><ol class="space-y-3 text-sm text-gray-700 dark:text-gray-300"><li>1. Open the repository on GitHub and go to <strong>Settings → Webhooks</strong>.</li><li>2. Add a webhook and paste the payload URL shown here.</li><li>3. Set the content type to <span class="font-mono">application/json</span> and paste the secret.</li><li>4. Select push events and save the webhook.</li></ol></div>
		</div>
	</div>
{/if}
