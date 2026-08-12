<script lang="ts">
	import { Check, CircleAlert, Copy, Eye, EyeOff, X } from '@lucide/svelte';
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

	let name        = '';
	let branch      = '';
	let imageRef    = '';
	let appPort     = 3000;
	let mainService = '';
	let resourceProfile: ResourceProfile = 'custom';
	let memoryMb    = 512;
	let cpuLimit    = 0.5;
	let composeFilePath      = '';
	let composeOverridePaths = '';
	let composeProfiles      = '';
	let composeWorkdir       = '';
	let staticFrontendPath   = '';
	let baseDirectory        = '';
	let serviceResourcesStr  = '{}';
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
		{ id: 'node-python',  title: 'Node/Python',       memoryMb: 256, cpuLimit: 0.35 },
		{ id: 'go-small',     title: 'Go small',          memoryMb: 128, cpuLimit: 0.2 },
		{ id: 'compose-main', title: 'Compose main',      memoryMb: 256, cpuLimit: 0.35 },
		{ id: 'static',       title: 'Static/no-runtime', memoryMb: 64,  cpuLimit: 0.1 },
		{ id: 'custom',       title: 'Custom',            memoryMb: 512, cpuLimit: 0.5 }
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
	                 memoryMb !== project.memoryLimitMb || cpuLimit !== project.cpuLimit);
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
		if (copiedResetTimer) {
			clearTimeout(copiedResetTimer);
		}
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
			if (project.deployMode === 'compose') {
				await loadComposeResources(project.id);
			}
			if (project.sourceType === 'git') {
				void inspectRepository(false, true).catch(() => undefined);
			}
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
		if (!force && requestKey === lastRepoInspectKey && !repoInspectError) {
			return true;
		}

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
			if (!branch.trim() && inspection.branch) {
				branch = inspection.branch;
			}
			repoTree = inspection.tree ?? [];
			repoTreeTruncated = inspection.treeTruncated ?? false;
			repoInspectMessage = `Repository validated on ${inspection.branch || branch || 'default branch'}`;
			lastRepoInspectKey = repositoryInspectionKey();
			if (showToast) {
				toast.success('Repository validated');
			}
			return true;
		} catch (err) {
			if (requestId !== repoInspectRequest) return false;
			const message = err instanceof Error ? err.message : 'Failed to inspect repository';
			repoInspectError = message;
			repoInspectMessage = '';
			repoTree = [];
			repoTreeTruncated = false;
			lastRepoInspectKey = '';
			if (showToast) {
				toast.error(message);
			}
			return false;
		} finally {
			if (requestId === repoInspectRequest) {
				inspectingRepo = false;
			}
		}
	}

	async function validateRepositoryBeforeSave() {
		if (!project || project.sourceType !== 'git') return true;
		if (!gitSourceChanged) return true;
		if (repositoryInspectionKey() === lastRepoInspectKey && !repoInspectError) {
			return true;
		}
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
			} catch (e) {
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
		return value
			.split(',')
			.map((entry) => entry.trim())
			.filter(Boolean);
	}

	function requestRegenerateSecret() {
		confirmRegenerateSecret = true;
	}

	function handleWindowKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && showWebhookHelp) {
			showWebhookHelp = false;
		}
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
				if (copiedResetTimer) {
					clearTimeout(copiedResetTimer);
				}
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
		<div class="surface h-48 animate-pulse"></div>
		<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
			<div class="surface h-64 animate-pulse"></div>
			<div class="surface h-64 animate-pulse"></div>
		</div>
	</div>
{:else if loadError || !project}
	<div class="surface overflow-hidden">
		<ErrorState title="Could not load settings" message={loadError || 'Project not found'} on:retry={() => void load()} />
	</div>
{:else if project}
	<div class="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
		<div class="space-y-4">
			<SectionPanel
				title="General"
				description="Source, routing, and runtime settings."
			>
				<div class="grid gap-4 sm:grid-cols-2">
					<div class="sm:col-span-2">
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="pname">Project name</label>
						<input id="pname" type="text" value={name} class="field w-full bg-gray-50 text-gray-500 dark:bg-gray-950 dark:text-gray-400" readonly />
						<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
							Project identity and subdomain are fixed after creation.
						</p>
					</div>
					{#if project.sourceType === 'registry'}
						<div class="sm:col-span-2">
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="imageRef">Container image</label>
							<input id="imageRef" type="text" bind:value={imageRef} placeholder="ghcr.io/example/app:latest" class="field w-full font-mono" />
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Deploy pulls this public OCI image. Changing the reference takes effect on the next deploy.</p>
						</div>
					{:else}
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="pbranch">Deploy branch</label>
							<input id="pbranch" type="text" value={branch} on:input={handleBranchInput} class="field w-full font-mono" />
						</div>
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="baseDirectory">Base directory</label>
							<input id="baseDirectory" type="text" value={baseDirectory} on:input={handleBaseDirectoryInput} placeholder="/" class="field w-full font-mono" />
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Deploy from a specific subdirectory instead of the repo root.</p>
						</div>
						<div class="sm:col-span-2 rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-xs dark:border-gray-800 dark:bg-gray-950/70">
							<div class="flex flex-wrap items-center justify-between gap-2">
								<div>
									{#if inspectingRepo}
										<p class="text-gray-600 dark:text-gray-300">Validating repository source...</p>
									{:else if repoInspectError}
										<p class="text-red-600 dark:text-red-300">{repoInspectError}</p>
									{:else if repoInspectMessage && !repoValidationStale}
										<p class="text-emerald-700 dark:text-emerald-300">{repoInspectMessage}</p>
									{:else}
										<p class="text-amber-700 dark:text-amber-300">Validate the repository before saving source changes.</p>
									{/if}
								</div>
								<ActionButton size="xs" variant="secondary" on:click={() => void inspectRepository(true, true)} loading={inspectingRepo} loadingLabel="Validating...">
									Validate source
								</ActionButton>
							</div>
							{#if repoDirectorySuggestions.length > 0}
								<div class="mt-2 flex flex-wrap gap-2">
									{#each repoDirectorySuggestions as entry}
										<button
											type="button"
											class="rounded-md border border-gray-200 bg-white px-2 py-1 font-mono text-[11px] text-gray-600 hover:border-gray-400 hover:text-gray-950 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:text-white"
											on:click={() => {
												baseDirectory = entry.path;
												clearRepositoryValidation();
											}}
										>
											{entry.path}
										</button>
									{/each}
								</div>
							{/if}
							{#if repoTreeTruncated}
								<p class="mt-2 text-gray-500 dark:text-gray-400">Repository tree is truncated; enter deeper paths manually if needed.</p>
							{/if}
						</div>
					{/if}
					{#if project.deployMode !== 'static'}
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="appPort">App port</label>
							<input id="appPort" type="number" min="1" max="65535" bind:value={appPort} class="field w-full font-mono" />
						</div>
					{/if}
					{#if project.deployMode === 'compose'}
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="mainService">Main service</label>
							<input id="mainService" type="text" bind:value={mainService} placeholder="app" class="field w-full font-mono" />
						</div>
					{/if}
					{#if project.sourceType === 'git' && (project.deployMode === 'compose' || project.deployMode === 'dockerfile')}
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="staticFrontendPath">Static Frontend Path</label>
							<input id="staticFrontendPath" type="text" bind:value={staticFrontendPath} placeholder="e.g. frontend" class="field w-full font-mono" />
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">If set, MyPaas builds and serves this directory statically alongside your backend.</p>
						</div>
					{/if}
				</div>
			</SectionPanel>

			{#if project.deployMode === 'compose'}
				<SectionPanel
					title="Compose configuration"
					description="Point MyPaas at a compose file anywhere in the repository, chain override files, and pick profiles. Leave path blank to auto-discover the top-ranked candidate."
				>
					<div class="grid gap-4 sm:grid-cols-2">
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeFilePath">Compose file path</label>
							<input
								id="composeFilePath"
								type="text"
								bind:value={composeFilePath}
								placeholder="auto-detect"
								class="field w-full font-mono"
							/>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Repo-relative, forward slashes only. e.g. <span class="font-mono">infra/docker-compose.yml</span>.</p>
						</div>
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeWorkdir">Working directory override</label>
							<input
								id="composeWorkdir"
								type="text"
								bind:value={composeWorkdir}
								placeholder="auto (parent of compose file)"
								class="field w-full font-mono"
							/>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Set only if build contexts or env files resolve against a different directory.</p>
						</div>
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeOverridePaths">Override files</label>
							<input
								id="composeOverridePaths"
								type="text"
								bind:value={composeOverridePaths}
								placeholder="docker-compose.prod.yml"
								class="field w-full font-mono"
							/>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Comma-separated, repo-relative. Applied before MyPaas' generated override.</p>
						</div>
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeProfiles">Profiles</label>
							<input
								id="composeProfiles"
								type="text"
								bind:value={composeProfiles}
								placeholder="app, worker"
								class="field w-full font-mono"
							/>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Comma-separated <span class="font-mono">COMPOSE_PROFILES</span> values.</p>
						</div>
					</div>
				</SectionPanel>
			{/if}

			<SectionPanel
				title="Resource limits"
				description="Default limits applied to the main service."
				contentClass="p-0"
			>
				<div class="grid gap-4 p-5 sm:grid-cols-2">
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="profile">Profile</label>
						<select id="profile" bind:value={resourceProfile} on:change={() => applyResourceProfile(resourceProfile)} class="field w-full">
							{#each resourceProfiles as profile}
								<option value={profile.id}>{profile.title} ({profile.memoryMb} MB / {profile.cpuLimit} CPU)</option>
							{/each}
						</select>
					</div>
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="mem">Memory</label>
						<select id="mem" bind:value={memoryMb} on:change={markCustomProfile} class="field w-full">
							{#each [64, 128, 256, 512, 1024, 2048] as m}
								<option value={m}>{m} MB</option>
							{/each}
						</select>
					</div>
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="cpu">CPU</label>
						<select id="cpu" bind:value={cpuLimit} on:change={markCustomProfile} class="field w-full">
							{#each [0.1, 0.2, 0.25, 0.35, 0.5, 1, 2] as c}
								<option value={c}>{c} core{c !== 1 ? 's' : ''}</option>
							{/each}
						</select>
					</div>
				</div>

				<div class="border-t border-gray-100 px-5 py-4 dark:border-gray-800">
					<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="service_resources">
						Other Services Limits (JSON)
					</label>
					<textarea
						id="service_resources"
						bind:value={serviceResourcesStr}
						rows="4"
						class="field w-full font-mono text-sm"
						placeholder='&#123;&#10;  "db": &#123;&#10;    "memoryLimitMb": 512,&#10;    "cpuLimit": 0.5&#10;  &#125;&#10;&#125;'
					></textarea>
					<p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
						Set memory and CPU limits for non-main services. Key is service name.
					</p>
					<div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
						<span class="font-medium text-gray-700 dark:text-gray-300">Example:</span>
						<pre class="mt-1 rounded-md border border-gray-100 bg-gray-50 p-2 text-[11px] text-gray-600 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-400">
&#123;
  "db": &#123;
    "memoryLimitMb": 256,
    "cpuLimit": 0.25
  &#125;
&#125;</pre>
					</div>
				</div>
				{#if settingsChanged}
					<div class="flex items-center justify-between gap-3 border-t border-gray-100 bg-gray-50/70 px-5 py-3 dark:border-gray-800 dark:bg-gray-900/70">
						<p class="text-xs text-gray-500 dark:text-gray-400">Unsaved project configuration changes.</p>
						<ActionButton
							variant="primary"
							on:click={handleSave}
							disabled={project.sourceType === 'git' && repoValidationStale && !repoInspectError}
							loading={savingSettings}
							loadingLabel="Saving..."
						>
							Save changes
						</ActionButton>
					</div>
				{/if}
			</SectionPanel>
		</div>

		<div class="space-y-4">
			{#if project.sourceType === 'git'}
			<SectionPanel
				title="Webhook"
				description="Use this for GitHub push deploys."
				contentClass="p-0"
			>
				<svelte:fragment slot="actions">
					<IconButton label="Webhook setup instructions" variant="brand" on:click={() => (showWebhookHelp = true)}>
						<CircleAlert class="h-4 w-4" aria-hidden="true" />
					</IconButton>
				</svelte:fragment>
				<div class="space-y-4 p-5">
					<div>
						<div class="mb-1 flex items-center justify-between">
							<p class="text-xs font-medium text-gray-600 dark:text-gray-300">Payload URL</p>
							<IconButton
								label={copiedTarget === 'webhook-url' ? 'Payload URL copied' : 'Copy payload URL'}
								variant={copiedTarget === 'webhook-url' ? 'brand' : 'ghost'}
								on:click={() => copyWebhookURL(project?.id ?? '')}
							>
								{#if copiedTarget === 'webhook-url'}
									<Check class="h-4 w-4" aria-hidden="true" />
								{:else}
									<Copy class="h-4 w-4" aria-hidden="true" />
								{/if}
							</IconButton>
						</div>
						<code class="block break-all rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300">
							{publicWebhookURL}
						</code>
					</div>
					<div>
						<div class="mb-1 flex items-center justify-between">
							<p class="text-xs font-medium text-gray-600 dark:text-gray-300">Secret</p>
							<div class="flex gap-3">
								<IconButton label={showWebhookSecret ? 'Hide webhook secret' : 'Show webhook secret'} variant="ghost" on:click={() => (showWebhookSecret = !showWebhookSecret)}>
									{#if showWebhookSecret}
										<EyeOff class="h-4 w-4" aria-hidden="true" />
									{:else}
										<Eye class="h-4 w-4" aria-hidden="true" />
									{/if}
								</IconButton>
								<IconButton
									label={copiedTarget === 'webhook-secret' ? 'Webhook secret copied' : 'Copy webhook secret'}
									variant={copiedTarget === 'webhook-secret' ? 'brand' : 'ghost'}
									on:click={() => copyText(project?.webhookSecret ?? '', 'Webhook secret copied', 'webhook-secret')}
								>
									{#if copiedTarget === 'webhook-secret'}
										<Check class="h-4 w-4" aria-hidden="true" />
									{:else}
										<Copy class="h-4 w-4" aria-hidden="true" />
									{/if}
								</IconButton>
							</div>
						</div>
						<code class="block break-all rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300">
							{showWebhookSecret ? project.webhookSecret : '••••••••••••••••••••••••••••••••'}
						</code>
					</div>
					{#if confirmRegenerateSecret}
						<div class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
							<p>Regenerating the secret invalidates existing GitHub webhook signatures.</p>
							<div class="mt-3 flex flex-wrap gap-2">
								<ActionButton variant="ghost" size="xs" on:click={() => (confirmRegenerateSecret = false)}>
									Cancel
								</ActionButton>
								<ActionButton variant="danger" size="xs" on:click={handleRegenerateSecret} loading={regeneratingSecret} loadingLabel="Regenerating...">
									Regenerate now
								</ActionButton>
							</div>
						</div>
					{:else}
						<ActionButton on:click={requestRegenerateSecret}>
							Regenerate secret
						</ActionButton>
					{/if}
				</div>
			</SectionPanel>
			{/if}

			{#if project.deployMode === 'compose'}
				<SectionPanel
					title="Compose resources"
					description="Tracked Docker resources for this Compose project."
					contentClass="p-0"
				>
					<div class="space-y-4 p-5">
						<div class="grid grid-cols-3 overflow-hidden rounded-md border border-gray-200 text-center dark:border-gray-800">
							<div class="border-r border-gray-200 px-3 py-2 dark:border-gray-800">
								<p class="text-lg font-semibold text-gray-950 dark:text-white">{composeResources?.containers ?? 0}</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">Containers</p>
							</div>
							<div class="border-r border-gray-200 px-3 py-2 dark:border-gray-800">
								<p class="text-lg font-semibold text-gray-950 dark:text-white">{composeResources?.volumes ?? 0}</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">Volumes</p>
							</div>
							<div class="px-3 py-2">
								<p class="text-lg font-semibold text-gray-950 dark:text-white">{composeResources?.networks ?? 0}</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">Networks</p>
							</div>
						</div>
						{#if composeResourceTotal > 0 && !project.activeDeploymentId}
							<p class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
								Compose resources exist but this project has no active deployment. Reset them before deploy if they are stale leftovers.
							</p>
						{/if}
						{#if composeResourceError}
							<div class="flex flex-col gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-200">
								<span>{composeResourceError}</span>
								<ActionButton variant="ghost" size="xs" on:click={() => loadComposeResources()}>
									Retry check
								</ActionButton>
							</div>
						{/if}
						<div class="flex gap-2">
							<ActionButton variant="secondary" on:click={() => loadComposeResources()} loading={loadingComposeResources} loadingLabel="Checking...">
								Check resources
							</ActionButton>
							<ActionButton variant="danger" on:click={requestResetComposeResources} disabled={composeResourceTotal === 0 || confirmResetComposeResources}>
								Reset
							</ActionButton>
						</div>
						{#if confirmResetComposeResources}
							<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-200">
								<p>This removes Compose containers, volumes, networks, route, and allocated port for this project.</p>
								<div class="mt-3 flex flex-wrap gap-2">
									<ActionButton variant="ghost" size="xs" on:click={() => (confirmResetComposeResources = false)}>
										Cancel
									</ActionButton>
									<ActionButton variant="danger" size="xs" on:click={handleResetComposeResources} loading={resettingComposeResources} loadingLabel="Resetting...">
										Reset now
									</ActionButton>
								</div>
							</div>
						{/if}
					</div>
				</SectionPanel>
			{/if}

			<section class="overflow-hidden rounded-lg border border-red-200 bg-white dark:border-red-900/60 dark:bg-gray-900">
				<div class="border-b border-red-100 px-5 py-4 dark:border-red-900/50">
					<h2 class="text-sm font-semibold text-red-700 dark:text-red-300">Danger zone</h2>
				</div>
				<div class="space-y-3 p-5">
					<p class="text-sm text-gray-600 dark:text-gray-400">
						Delete this project, stop containers, remove routing, and release ports.
					</p>
					<input
						type="text"
						bind:value={deleteInput}
						placeholder={project.name}
						class="field w-full border-red-300 focus:border-red-600 focus:ring-red-600 dark:border-red-900"
					/>
					<ActionButton
						variant="danger"
						on:click={handleDelete}
						disabled={deleteInput !== project.name}
						loading={deletingProject}
						loadingLabel="Deleting..."
						full
					>
						Delete project
					</ActionButton>
				</div>
			</section>
		</div>
	</div>
{/if}

{#if showWebhookHelp && project && project.sourceType === 'git'}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4"
	>
		<button
			type="button"
			class="absolute inset-0 cursor-default bg-gray-950/45 backdrop-blur-sm"
			aria-label="Close webhook setup"
			on:click={() => (showWebhookHelp = false)}
		></button>
		<div
			class="surface relative max-h-[90vh] w-full max-w-2xl overflow-hidden shadow-xl shadow-gray-950/20"
			role="dialog"
			aria-modal="true"
			aria-labelledby="webhook-help-title"
			tabindex="-1"
		>
			<div class="panel-header flex items-start justify-between gap-3">
				<div class="min-w-0">
					<h2 id="webhook-help-title" class="text-sm font-semibold text-gray-950 dark:text-white">GitHub webhook setup</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Configure push deploys for the selected repository.</p>
				</div>
				<IconButton label="Close webhook setup" variant="ghost" on:click={() => (showWebhookHelp = false)}>
					<X class="h-4 w-4" aria-hidden="true" />
				</IconButton>
			</div>

			<div class="max-h-[calc(90vh-5rem)] space-y-5 overflow-y-auto p-5">
				<div class="grid gap-3 sm:grid-cols-[8rem_minmax(0,1fr)]">
					<span class="metric-label">Payload URL</span>
					<div class="flex min-w-0 items-start gap-2">
						<code class="min-w-0 flex-1 break-all rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300">
							{publicWebhookURL}
						</code>
						<IconButton
							label={copiedTarget === 'webhook-url' ? 'Payload URL copied' : 'Copy payload URL'}
							variant={copiedTarget === 'webhook-url' ? 'brand' : 'default'}
							on:click={() => copyWebhookURL(project?.id ?? '')}
						>
							{#if copiedTarget === 'webhook-url'}
								<Check class="h-4 w-4" aria-hidden="true" />
							{:else}
								<Copy class="h-4 w-4" aria-hidden="true" />
							{/if}
						</IconButton>
					</div>

					<span class="metric-label">Secret</span>
					<div class="flex min-w-0 items-start gap-2">
						<code class="min-w-0 flex-1 break-all rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300">
							{showWebhookSecret ? project.webhookSecret : '••••••••••••••••••••••••••••••••'}
						</code>
						<IconButton
							label={copiedTarget === 'webhook-secret' ? 'Webhook secret copied' : 'Copy webhook secret'}
							variant={copiedTarget === 'webhook-secret' ? 'brand' : 'default'}
							on:click={() => copyText(project?.webhookSecret ?? '', 'Webhook secret copied', 'webhook-secret')}
						>
							{#if copiedTarget === 'webhook-secret'}
								<Check class="h-4 w-4" aria-hidden="true" />
							{:else}
								<Copy class="h-4 w-4" aria-hidden="true" />
							{/if}
						</IconButton>
					</div>
				</div>

				<ol class="space-y-3 text-sm text-gray-700 dark:text-gray-300">
					<li class="flex gap-3">
						<span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-gray-900 text-xs font-semibold text-white dark:bg-gray-100 dark:text-gray-950">1</span>
						<span>Open the GitHub repository, then go to <span class="font-medium text-gray-950 dark:text-white">Settings</span> and <span class="font-medium text-gray-950 dark:text-white">Webhooks</span>.</span>
					</li>
					<li class="flex gap-3">
						<span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-gray-900 text-xs font-semibold text-white dark:bg-gray-100 dark:text-gray-950">2</span>
						<span>Choose <span class="font-medium text-gray-950 dark:text-white">Add webhook</span>, paste the payload URL, and set content type to <span class="font-mono">application/json</span>.</span>
					</li>
					<li class="flex gap-3">
						<span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-gray-900 text-xs font-semibold text-white dark:bg-gray-100 dark:text-gray-950">3</span>
						<span>Paste the webhook secret, keep <span class="font-medium text-gray-950 dark:text-white">Just the push event</span> selected, and leave the webhook active.</span>
					</li>
					<li class="flex gap-3">
						<span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-gray-900 text-xs font-semibold text-white dark:bg-gray-100 dark:text-gray-950">4</span>
						<span>Save it. MyPaas deploys only when the push targets the configured branch: <span class="font-mono">{project.branch}</span>.</span>
					</li>
				</ol>

				<div class="rounded-md border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-800 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-200">
					<p class="font-medium">Automatic deploy without webhook?</p>
					<p class="mt-1">GitHub does not push commit events to MyPaas unless MyPaas is registered through a webhook or GitHub App. Polling the GitHub API can work, but it is slower, noisier, and needs extra token scope.</p>
				</div>
			</div>
		</div>
	</div>
{/if}
