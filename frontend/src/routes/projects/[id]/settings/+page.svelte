<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import ActionButton from '$components/ActionButton.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import IconButton from '$components/IconButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { ComposeResourceSummary, Project, ResourceProfile } from '$types';
	import { projectHost, webhookURL } from '$lib/utils/urls';

	let project: Project | null = null;
	let composeResources: ComposeResourceSummary | null = null;
	let loading = true;
	let loadError = '';
	let composeResourceError = '';

	let name        = '';
	let branch      = '';
	let appPort     = 3000;
	let resourceProfile: ResourceProfile = 'custom';
	let memoryMb    = 512;
	let cpuLimit    = 0.5;
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

	$: nameChanged = project && (name !== project.name || branch !== project.branch ||
	                 (project.deployMode !== 'static' && appPort !== project.appPort) ||
	                 resourceProfile !== project.resourceProfile ||
	                 memoryMb !== project.memoryLimitMb || cpuLimit !== project.cpuLimit);
	$: changedProjectHost = projectHost(name || 'your-app', $page.url.hostname);
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
			project = await api.projects.get($page.params.id);
			name = project.name;
			branch = project.branch;
			appPort = project.appPort;
			resourceProfile = project.resourceProfile;
			memoryMb = project.memoryLimitMb;
			cpuLimit = project.cpuLimit;
			if (project.deployMode === 'compose') {
				await loadComposeResources(project.id);
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

	async function handleSave() {
		if (!project || savingSettings) return;
		savingSettings = true;
		try {
			project = await api.projects.update(project.id, {
				name,
				branch,
				resourceProfile,
				appPort: Number(appPort),
				memoryLimitMb: Number(memoryMb),
				cpuLimit: Number(cpuLimit)
			});
			toast.success('Settings saved');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save settings');
		} finally {
			savingSettings = false;
		}
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
	<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
		<div class="space-y-4">
			<SectionPanel
				title="General"
				description="Routing and deployment branch settings."
			>
				<div class="grid gap-4 sm:grid-cols-2">
					<div class="sm:col-span-2">
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="pname">Project name</label>
						<input id="pname" type="text" bind:value={name} class="field w-full" />
						{#if name !== project.name}
							<p class="mt-1 text-xs text-amber-600 dark:text-amber-300">
								Subdomain will change to <span class="font-mono">{changedProjectHost}</span>
							</p>
						{/if}
					</div>
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="pbranch">Deploy branch</label>
						<input id="pbranch" type="text" bind:value={branch} class="field w-full font-mono" />
					</div>
					{#if project.deployMode !== 'static'}
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="appPort">App port</label>
							<input id="appPort" type="number" min="1" max="65535" bind:value={appPort} class="field w-full font-mono" />
						</div>
					{/if}
				</div>
			</SectionPanel>

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
				{#if nameChanged}
					<div class="flex items-center justify-between gap-3 border-t border-gray-100 bg-gray-50/70 px-5 py-3 dark:border-gray-800 dark:bg-gray-900/70">
						<p class="text-xs text-gray-500 dark:text-gray-400">Unsaved project configuration changes.</p>
						<ActionButton variant="primary" on:click={handleSave} loading={savingSettings} loadingLabel="Saving...">
							Save changes
						</ActionButton>
					</div>
				{/if}
			</SectionPanel>
		</div>

		<div class="space-y-4">
			<SectionPanel
				title="Webhook"
				description="Use this for GitHub push deploys."
				contentClass="p-0"
			>
				<svelte:fragment slot="actions">
					<IconButton label="Webhook setup instructions" variant="brand" on:click={() => (showWebhookHelp = true)}>
						<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M11.25 11.25h1.5v6h-1.5zM12 7.5h.01" />
							<path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
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
									<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
										<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
									</svg>
								{:else}
									<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
										<path stroke-linecap="round" stroke-linejoin="round" d="M8 7h10a2 2 0 012 2v10a2 2 0 01-2 2H8a2 2 0 01-2-2V9a2 2 0 012-2z" />
										<path stroke-linecap="round" stroke-linejoin="round" d="M4 15H3a2 2 0 01-2-2V5a2 2 0 012-2h10a2 2 0 012 2v1" />
									</svg>
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
										<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M3 3l18 18" />
											<path stroke-linecap="round" stroke-linejoin="round" d="M10.6 10.6a2 2 0 002.8 2.8" />
											<path stroke-linecap="round" stroke-linejoin="round" d="M9.9 4.2A10.7 10.7 0 0112 4c5 0 8.5 4 10 8a15.1 15.1 0 01-3.1 4.7M6.6 6.6A14.6 14.6 0 002 12c1.5 4 5 8 10 8a10.8 10.8 0 005.4-1.5" />
										</svg>
									{:else}
										<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M2 12s3.5-8 10-8 10 8 10 8-3.5 8-10 8-10-8-10-8z" />
											<path stroke-linecap="round" stroke-linejoin="round" d="M12 15a3 3 0 100-6 3 3 0 000 6z" />
										</svg>
									{/if}
								</IconButton>
								<IconButton
									label={copiedTarget === 'webhook-secret' ? 'Webhook secret copied' : 'Copy webhook secret'}
									variant={copiedTarget === 'webhook-secret' ? 'brand' : 'ghost'}
									on:click={() => copyText(project?.webhookSecret ?? '', 'Webhook secret copied', 'webhook-secret')}
								>
									{#if copiedTarget === 'webhook-secret'}
										<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
										</svg>
									{:else}
										<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M8 7h10a2 2 0 012 2v10a2 2 0 01-2 2H8a2 2 0 01-2-2V9a2 2 0 012-2z" />
											<path stroke-linecap="round" stroke-linejoin="round" d="M4 15H3a2 2 0 01-2-2V5a2 2 0 012-2h10a2 2 0 012 2v1" />
										</svg>
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

{#if showWebhookHelp && project}
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
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 6l12 12M18 6L6 18" />
					</svg>
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
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
								</svg>
							{:else}
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M8 7h10a2 2 0 012 2v10a2 2 0 01-2 2H8a2 2 0 01-2-2V9a2 2 0 012-2z" />
									<path stroke-linecap="round" stroke-linejoin="round" d="M4 15H3a2 2 0 01-2-2V5a2 2 0 012-2h10a2 2 0 012 2v1" />
								</svg>
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
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
								</svg>
							{:else}
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M8 7h10a2 2 0 012 2v10a2 2 0 01-2 2H8a2 2 0 01-2-2V9a2 2 0 012-2z" />
									<path stroke-linecap="round" stroke-linejoin="round" d="M4 15H3a2 2 0 01-2-2V5a2 2 0 012-2h10a2 2 0 012 2v1" />
								</svg>
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
