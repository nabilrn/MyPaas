<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { ComposeResourceSummary, Project, ResourceProfile } from '$types';
	import { projectHost, webhookURL } from '$lib/utils/urls';

	let project: Project | null = null;
	let composeResources: ComposeResourceSummary | null = null;

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

	onMount(load);

	async function load() {
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
	}

	async function loadComposeResources(projectId = project?.id) {
		if (!projectId) return;
		loadingComposeResources = true;
		try {
			composeResources = await api.projects.composeResources(projectId);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load Compose resources');
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

	async function handleRegenerateSecret() {
		if (!project || regeneratingSecret || !window.confirm('Regenerate webhook secret? Existing GitHub webhook signatures will stop working until you update the secret.')) return;
		regeneratingSecret = true;
		try {
			const result = await api.projects.regenerateWebhookSecret(project.id);
			project = { ...project, webhookSecret: result.webhookSecret };
			showWebhookSecret = true;
			toast.success('Webhook secret regenerated');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to regenerate webhook secret');
		} finally {
			regeneratingSecret = false;
		}
	}

	async function handleResetComposeResources() {
		if (!project || resettingComposeResources) return;
		if (!window.confirm('Reset Compose resources for this project? Containers, Compose volumes, networks, route, and allocated port will be removed.')) return;
		resettingComposeResources = true;
		try {
			await api.projects.resetComposeResources(project.id);
			project = { ...project, status: 'stopped', allocatedPort: null, activeDeploymentId: null };
			await loadComposeResources(project.id);
			toast.success('Compose resources reset');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to reset Compose resources');
		} finally {
			resettingComposeResources = false;
		}
	}

	function copyWebhookURL(projectId: string) {
		copyText(webhookURL(projectId, $page.url.origin), 'Webhook URL copied');
	}

	function copyText(value: string, successMessage: string) {
		navigator.clipboard?.writeText(value)
			.then(() => toast.success(successMessage))
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

<svelte:head>
	<title>Settings · MyPaas</title>
</svelte:head>

{#if !project}
	<div class="surface h-48 animate-pulse"></div>
{:else}
	<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
		<div class="space-y-4">
			<section class="surface overflow-hidden">
				<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">General</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Routing and deployment branch settings.</p>
				</div>
				<div class="grid gap-4 p-5 sm:grid-cols-2">
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
			</section>

			<section class="surface overflow-hidden">
				<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Resource limits</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Default limits applied to the main service.</p>
				</div>
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
			</section>
		</div>

		<div class="space-y-4">
			<section class="surface overflow-hidden">
				<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Webhook</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Use this for GitHub push deploys.</p>
				</div>
				<div class="space-y-4 p-5">
					<div>
						<div class="mb-1 flex items-center justify-between">
							<p class="text-xs font-medium text-gray-600 dark:text-gray-300">Payload URL</p>
							<button on:click={() => copyWebhookURL(project?.id ?? '')} class="text-xs font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">
								Copy
							</button>
						</div>
						<code class="block break-all rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300">
							{publicWebhookURL}
						</code>
					</div>
					<div>
						<div class="mb-1 flex items-center justify-between">
							<p class="text-xs font-medium text-gray-600 dark:text-gray-300">Secret</p>
							<div class="flex gap-3">
								<button type="button" on:click={() => (showWebhookSecret = !showWebhookSecret)} class="text-xs font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">
									{showWebhookSecret ? 'Hide' : 'Show'}
								</button>
								<button type="button" on:click={() => copyText(project?.webhookSecret ?? '', 'Webhook secret copied')} class="text-xs font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">
									Copy
								</button>
							</div>
						</div>
						<code class="block break-all rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300">
							{showWebhookSecret ? project.webhookSecret : '••••••••••••••••••••••••••••••••'}
						</code>
					</div>
					<ActionButton on:click={handleRegenerateSecret} loading={regeneratingSecret} loadingLabel="Regenerating...">
						Regenerate secret
					</ActionButton>
				</div>
			</section>

			{#if project.deployMode === 'compose'}
				<section class="surface overflow-hidden">
					<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Compose resources</h2>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Tracked Docker resources for this Compose project.</p>
					</div>
					<div class="space-y-4 p-5">
						<div class="grid grid-cols-3 gap-2 text-center">
							<div class="rounded-md border border-gray-200 px-3 py-2 dark:border-gray-800">
								<p class="text-lg font-semibold text-gray-950 dark:text-white">{composeResources?.containers ?? 0}</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">Containers</p>
							</div>
							<div class="rounded-md border border-gray-200 px-3 py-2 dark:border-gray-800">
								<p class="text-lg font-semibold text-gray-950 dark:text-white">{composeResources?.volumes ?? 0}</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">Volumes</p>
							</div>
							<div class="rounded-md border border-gray-200 px-3 py-2 dark:border-gray-800">
								<p class="text-lg font-semibold text-gray-950 dark:text-white">{composeResources?.networks ?? 0}</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">Networks</p>
							</div>
						</div>
						{#if composeResourceTotal > 0 && !project.activeDeploymentId}
							<p class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
								Compose resources exist but this project has no active deployment. Reset them before deploy if they are stale leftovers.
							</p>
						{/if}
						<div class="flex gap-2">
							<ActionButton variant="secondary" on:click={() => loadComposeResources()} loading={loadingComposeResources} loadingLabel="Checking...">
								Check resources
							</ActionButton>
							<ActionButton variant="danger" on:click={handleResetComposeResources} disabled={composeResourceTotal === 0} loading={resettingComposeResources} loadingLabel="Resetting...">
								Reset
							</ActionButton>
						</div>
					</div>
				</section>
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
