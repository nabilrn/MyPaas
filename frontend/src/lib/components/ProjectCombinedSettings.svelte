<script lang="ts">
	import { Check, Copy, Eye, EyeOff, RefreshCw, X } from '@lucide/svelte';
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import ConfirmActionDialog from '$components/ConfirmActionDialog.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import IconButton from '$components/IconButton.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';
	import ProjectEffectiveConfiguration from '$components/ProjectEffectiveConfiguration.svelte';
	import SelectMenu from '$components/SelectMenu.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { ComposeResourceSummary, Project, RepoTreeEntry, ResourceProfile } from '$types';
	import { projectURL, webhookURL } from '$lib/utils/urls';

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
	let savingSource = false;
	let savingResources = false;
	let regeneratingSecret = false;
	let confirmRegenerateSecret = false;
	let showWebhookSecret = false;
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
	$: effectivePublicURL = project ? projectURL(project.subdomain, $page.url.protocol, $page.url.hostname) : '';
	$: publicWebhookURL = project ? webhookURL(project.id, $page.url.origin) : '';
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
			if (project.sourceType === 'git') void inspectRepository(false, true);
		} catch (error) {
			loadError = error instanceof Error ? error.message : 'Failed to load project settings';
		} finally {
			loading = false;
		}
	}

	async function loadComposeResources(projectId = project?.id) {
		if (!projectId) return;
		composeResourceError = '';
		try {
			composeResources = await api.projects.composeResources(projectId);
		} catch (error) {
			composeResourceError = error instanceof Error ? error.message : 'Failed to load runtime resources';
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
		} catch (error) {
			if (requestId !== repoInspectRequest) return false;
			repoInspectError = error instanceof Error ? error.message : 'Could not check repository';
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

	function splitCommaList(value: string) {
		return value.split(',').map((entry) => entry.trim()).filter(Boolean);
	}

	async function saveSource() {
		if (!project || savingSource) return;
		savingSource = true;
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
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to save source settings');
		} finally {
			savingSource = false;
		}
	}

	async function saveResources() {
		if (!project || savingResources) return;
		savingResources = true;
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
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to save resource settings');
		} finally {
			savingResources = false;
		}
	}

	async function regenerateSecret() {
		if (!project || regeneratingSecret) return;
		regeneratingSecret = true;
		try {
			const result = await api.projects.regenerateWebhookSecret(project.id);
			project = { ...project, webhookSecret: result.webhookSecret };
			showWebhookSecret = true;
			confirmRegenerateSecret = false;
			toast.success('Webhook secret regenerated');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to regenerate webhook secret');
		} finally {
			regeneratingSecret = false;
		}
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
</script>

<svelte:window on:keydown={(event) => { if (event.key === 'Escape' && showWebhookHelp) showWebhookHelp = false; }} />

{#if loading}
	<div class="flex min-h-64 items-center justify-center"><LoadingIndicator label="Loading settings" /></div>
{:else if loadError || !project}
	<ErrorState title="Could not load settings" message={loadError || 'Project not found'} on:retry={() => void load()} />
{:else if project}
	<div class="project-settings-workspace w-full">
		<div class="border-b border-[color:var(--workspace-divider)] px-5 pt-4 pb-3">
			<h1 class="text-lg font-semibold text-gray-950 dark:text-white">Settings</h1>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Project identity, deployment source, resources, and webhook configuration.</p>
		</div>

		<section class="border-b border-[color:var(--workspace-divider)]">
			<div class="px-4 py-2.5"><h2 class="text-sm font-semibold text-gray-950 dark:text-white">General</h2></div>
			<div class="border-t border-[color:var(--workspace-divider)] px-4"><ProjectEffectiveConfiguration {project} publicUrl={effectivePublicURL} /></div>
		</section>

		<section class="border-b border-[color:var(--workspace-divider)]">
			<div class="px-4 py-2.5"><h2 class="text-sm font-semibold text-gray-950 dark:text-white">Source</h2><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">What MyPaaS deploys and where it is found.</p></div>
			<div class="border-t border-[color:var(--workspace-divider)]">
				{#if project.sourceType === 'registry'}
					<div class="grid gap-3 px-4 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-center"><label class="text-sm text-gray-500 dark:text-gray-400" for="imageRef">Container image</label><input id="imageRef" type="text" bind:value={imageRef} class="field w-full max-w-xl font-mono" /></div>
				{:else}
					<div class="divide-y divide-[color:var(--workspace-divider)]">
						<div class="grid gap-3 px-4 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-center"><p class="text-sm text-gray-500 dark:text-gray-400">Repository</p><p class="min-w-0 truncate font-mono text-sm text-gray-950 dark:text-white" title={project.repoUrl}>{project.repoUrl}</p></div>
						<div class="grid gap-3 px-4 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-center"><span class="text-sm text-gray-500 dark:text-gray-400">Branch</span><div class="max-w-xl"><SelectMenu value={branch} options={branchOptions} ariaLabel="Deployment branch" disabled={inspectingRepo || branchOptions.length === 0} on:change={(event) => handleBranchChange(event.detail)} /></div></div>
						<div class="grid gap-3 px-4 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-center"><span class="text-sm text-gray-500 dark:text-gray-400">Base directory</span><div class="max-w-xl"><SelectMenu value={baseDirectory} options={baseDirectoryOptions} ariaLabel="Base directory" disabled={inspectingRepo} on:change={(event) => handleBaseDirectoryChange(event.detail)} /></div></div>
					</div>
				{/if}

				{#if repoInspectError}<div class="mx-4 my-3 alert-danger flex-wrap items-center justify-between gap-3"><span class="min-w-0 flex-1">{repoInspectError}</span><ActionButton variant="ghost" size="xs" on:click={() => void inspectRepository(true, true)} loading={inspectingRepo} loadingLabel="Checking"><RefreshCw slot="icon" class="h-3.5 w-3.5" />Retry</ActionButton></div>{/if}

				<details class="border-t border-[color:var(--workspace-divider)] px-4 py-3">
					<summary class="app-focus cursor-pointer select-none text-sm font-medium text-gray-700 dark:text-gray-300">Advanced source settings</summary>
					<div class="mt-4 grid max-w-4xl gap-4 sm:grid-cols-2">
						{#if project.deployMode !== 'static'}<div><label class="field-label" for="appPort">App port</label><input id="appPort" type="number" min="1" max="65535" bind:value={appPort} class="field w-full max-w-xs font-mono" /></div>{/if}
						{#if project.deployMode === 'compose'}
							<div><label class="field-label" for="mainService">Main service</label><input id="mainService" type="text" bind:value={mainService} class="field w-full max-w-sm font-mono" /></div>
							<div><label class="field-label" for="composeFilePath">Compose file</label><input id="composeFilePath" type="text" bind:value={composeFilePath} class="field w-full max-w-sm font-mono" /></div>
							<div><label class="field-label" for="composeWorkdir">Working directory</label><input id="composeWorkdir" type="text" bind:value={composeWorkdir} class="field w-full max-w-sm font-mono" /></div>
							<div><label class="field-label" for="composeOverridePaths">Override files</label><input id="composeOverridePaths" type="text" bind:value={composeOverridePaths} class="field w-full max-w-sm font-mono" /></div>
							<div><label class="field-label" for="composeProfiles">Profiles</label><input id="composeProfiles" type="text" bind:value={composeProfiles} class="field w-full max-w-sm font-mono" /></div>
						{/if}
						{#if project.sourceType === 'git' && (project.deployMode === 'compose' || project.deployMode === 'dockerfile')}<div><label class="field-label" for="staticFrontendPath">Static frontend path</label><input id="staticFrontendPath" type="text" bind:value={staticFrontendPath} class="field w-full max-w-sm font-mono" /></div>{/if}
					</div>
				</details>
				<div class="flex flex-wrap items-center justify-between gap-3 border-t border-[color:var(--workspace-divider)] px-4 py-3"><p class="text-xs text-gray-500 dark:text-gray-400">Changes take effect on the next deployment.</p>{#if sourceChanged}<ActionButton variant="primary" size="sm" on:click={saveSource} loading={savingSource} loadingLabel="Saving">Save source</ActionButton>{/if}</div>
			</div>
		</section>

		<section class="border-b border-[color:var(--workspace-divider)]">
			<div class="px-4 py-2.5"><h2 class="text-sm font-semibold text-gray-950 dark:text-white">Resources</h2><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">CPU and memory allocation for this project.</p></div>
			<div class="grid border-t border-[color:var(--workspace-divider)] lg:grid-cols-[minmax(20rem,0.85fr)_minmax(0,1.15fr)]">
				<div class="px-4 py-3 lg:border-r lg:border-[color:var(--workspace-divider)]">
					<label class="field-label">Resource profile</label>
					<div class="max-w-md"><SelectMenu value={resourceProfile} options={resourceProfileOptions} ariaLabel="Resource profile" on:change={(event) => applyResourceProfile(event.detail as ResourceProfile)} /></div>
					{#if resourceProfile === 'custom'}
						<div class="mt-4 grid max-w-md gap-3 sm:grid-cols-2"><div><label class="field-label" for="mem">Memory (MB)</label><input id="mem" type="number" min="64" max="32768" step="1" bind:value={memoryMb} on:input={markCustomProfile} class="field w-full" /></div><div><label class="field-label" for="cpu">CPU</label><input id="cpu" type="number" min="0.1" max="32" step="0.05" bind:value={cpuLimit} on:input={markCustomProfile} class="field w-full" /></div></div>
					{:else}
						<div class="mt-4 grid max-w-md grid-cols-2 gap-4"><div><p class="text-xs text-gray-500 dark:text-gray-400">Memory</p><p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">{memoryMb} MB</p></div><div><p class="text-xs text-gray-500 dark:text-gray-400">CPU</p><p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">{formatCpu(cpuLimit)} CPU</p></div></div>
					{/if}
				</div>

				<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 lg:border-t-0">
					{#if project.deployMode === 'compose'}
						<div class="flex items-start justify-between gap-3"><div><p class="text-sm font-medium text-gray-950 dark:text-white">Runtime resources</p><p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{composeResources?.containers ?? 0} container{(composeResources?.containers ?? 0) === 1 ? '' : 's'} · {composeResources?.volumes ?? 0} volume{(composeResources?.volumes ?? 0) === 1 ? '' : 's'} · {composeResources?.networks ?? 0} network{(composeResources?.networks ?? 0) === 1 ? '' : 's'}</p></div><IconButton label="Refresh runtime resources" variant="secondary" on:click={() => void loadComposeResources()}><RefreshCw class="h-4 w-4" /></IconButton></div>
						{#if composeResourceError}<div class="mt-3 alert-danger">{composeResourceError}</div>{/if}
						<details class="mt-4 border-t border-[color:var(--workspace-divider)] pt-3"><summary class="app-focus cursor-pointer select-none text-sm font-medium text-gray-700 dark:text-gray-300">Advanced resource limits</summary><div class="mt-3"><label class="field-label" for="service_resources">Other services (JSON)</label><textarea id="service_resources" bind:value={serviceResourcesStr} rows="5" class="field w-full max-w-2xl font-mono text-sm"></textarea></div></details>
					{:else}
						<p class="text-sm font-medium text-gray-950 dark:text-white">Single runtime</p><p class="mt-1 text-xs text-gray-500 dark:text-gray-400">The selected profile controls this project's container allocation.</p>
					{/if}
				</div>
			</div>
			<div class="flex flex-wrap items-center justify-between gap-3 border-t border-[color:var(--workspace-divider)] px-4 py-3"><p class="text-xs text-gray-500 dark:text-gray-400">New limits apply after the next deployment.</p>{#if resourcesChanged}<ActionButton variant="primary" size="sm" on:click={saveResources} loading={savingResources} loadingLabel="Saving">Save resources</ActionButton>{/if}</div>
		</section>

		<section class="border-b border-[color:var(--workspace-divider)]">
			<div class="px-4 py-2.5"><h2 class="text-sm font-semibold text-gray-950 dark:text-white">Webhook</h2><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Deploy on GitHub push events.</p></div>
			{#if project.sourceType !== 'git'}
				<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3"><div class="alert-neutral">Webhook deployment is available only for projects using a Git repository source.</div></div>
			{:else}
				<div class="divide-y divide-[color:var(--workspace-divider)] border-t border-[color:var(--workspace-divider)]">
					<div class="grid gap-3 px-4 py-3 sm:grid-cols-[9rem_minmax(0,1fr)_auto] sm:items-center"><p class="text-sm text-gray-500 dark:text-gray-400">Payload URL</p><p class="min-w-0 break-all font-mono text-sm text-gray-950 dark:text-white">{publicWebhookURL}</p><IconButton label={copiedTarget === 'webhook-url' ? 'Payload URL copied' : 'Copy payload URL'} variant="ghost" on:click={() => copyText(publicWebhookURL, 'Webhook URL copied', 'webhook-url')}>{#if copiedTarget === 'webhook-url'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton></div>
					<div class="grid gap-3 px-4 py-3 sm:grid-cols-[9rem_minmax(0,1fr)_auto] sm:items-center"><p class="text-sm text-gray-500 dark:text-gray-400">Secret</p><p class="min-w-0 break-all font-mono text-sm text-gray-950 dark:text-white">{showWebhookSecret ? project.webhookSecret : '••••••••••••••••'}</p><div class="flex items-center gap-1"><IconButton label={showWebhookSecret ? 'Hide webhook secret' : 'Reveal webhook secret'} variant="ghost" on:click={() => (showWebhookSecret = !showWebhookSecret)}>{#if showWebhookSecret}<EyeOff class="h-4 w-4" />{:else}<Eye class="h-4 w-4" />{/if}</IconButton><IconButton label={copiedTarget === 'webhook-secret' ? 'Webhook secret copied' : 'Copy webhook secret'} variant="ghost" on:click={() => copyText(project?.webhookSecret ?? '', 'Webhook secret copied', 'webhook-secret')}>{#if copiedTarget === 'webhook-secret'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton></div></div>
				</div>
				<div class="flex flex-wrap items-center gap-2 border-t border-[color:var(--workspace-divider)] px-4 py-3"><ActionButton variant="secondary" size="sm" on:click={() => (showWebhookHelp = true)}>Setup guide</ActionButton><ActionButton variant="ghost" size="sm" on:click={() => (confirmRegenerateSecret = true)}>Regenerate secret</ActionButton></div>
			{/if}
		</section>
	</div>
{/if}

<ConfirmActionDialog
	open={confirmRegenerateSecret}
	title="Regenerate webhook secret?"
	description="GitHub deliveries using the current secret will fail until the webhook is updated."
	confirmLabel="Regenerate secret"
	busyLabel="Regenerating"
	variant="danger"
	busy={regeneratingSecret}
	on:cancel={() => (confirmRegenerateSecret = false)}
	on:confirm={regenerateSecret}
>
	<p>After regeneration, copy the new secret into the GitHub webhook configuration.</p>
</ConfirmActionDialog>

{#if showWebhookHelp && project?.sourceType === 'git'}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button type="button" class="absolute inset-0 cursor-default bg-gray-950/45" aria-label="Close webhook setup" on:click={() => (showWebhookHelp = false)}></button>
		<div class="overlay relative max-h-[90vh] w-full max-w-2xl overflow-hidden" role="dialog" aria-modal="true" aria-labelledby="webhook-help-title" tabindex="-1">
			<div class="panel-header flex items-start justify-between gap-3"><div><h2 id="webhook-help-title" class="panel-title">GitHub webhook setup</h2><p class="panel-description">Connect GitHub pushes to this project.</p></div><IconButton label="Close webhook setup" variant="ghost" on:click={() => (showWebhookHelp = false)}><X class="h-4 w-4" /></IconButton></div>
			<div class="max-h-[calc(90vh-5rem)] space-y-4 overflow-y-auto p-4"><ol class="space-y-3 text-sm text-gray-700 dark:text-gray-300"><li>1. Open the repository on GitHub and go to <strong>Settings → Webhooks</strong>.</li><li>2. Add a webhook and paste the payload URL shown here.</li><li>3. Set the content type to <span class="font-mono">application/json</span> and paste the secret.</li><li>4. Select push events and save the webhook.</li></ol></div>
		</div>
	</div>
{/if}
