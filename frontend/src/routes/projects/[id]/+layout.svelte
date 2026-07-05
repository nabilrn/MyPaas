<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import StatusBadge from '$components/StatusBadge.svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { Project } from '$types';
	import { projectHost, projectURL } from '$lib/utils/urls';

	let project: Project | null = null;
	let loading = true;
	let error = '';
	let pendingAction: 'stop' | 'restart' | 'deploy' | null = null;

	const tabs = [
		{ label: 'Overview',     href: '' },
		{ label: 'Deployments',  href: '/deployments' },
		{ label: 'Logs',         href: '/logs' },
		{ label: 'Metrics',      href: '/metrics' },
		{ label: 'Environment',  href: '/env' },
		{ label: 'Settings',     href: '/settings' }
	];

	$: base     = `/projects/${$page.params.id}`;
	$: pathname = $page.url.pathname;
	$: activeHref = tabs.slice().reverse().find((t) => pathname === base + t.href)?.href ?? '';
	$: publicProjectHost = project ? projectHost(project.subdomain, $page.url.hostname) : '';
	$: publicProjectURL = project ? projectURL(project.subdomain, $page.url.protocol, $page.url.hostname) : '';

	onMount(loadProject);

	async function loadProject(background = false) {
		if (!background || !project) {
			loading = true;
		}
		error = '';
		try {
			project = await api.projects.get($page.params.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load project';
		} finally {
			if (!background || !project) {
				loading = false;
			}
		}
	}

	async function handleStop() {
		if (!project || pendingAction) return;
		pendingAction = 'stop';
		try {
			await api.projects.stop(project.id);
			toast.success(`Stopped ${project.name}`);
			await loadProject(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to stop project');
		} finally {
			pendingAction = null;
		}
	}

	async function handleRestart() {
		if (!project || pendingAction) return;
		pendingAction = 'restart';
		try {
			await api.projects.restart(project.id);
			toast.success(`Restarted ${project.name}`);
			await loadProject(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to restart project');
		} finally {
			pendingAction = null;
		}
	}

	async function handleDeploy() {
		if (!project || pendingAction) return;
		pendingAction = 'deploy';
		try {
			await api.projects.deploy(project.id);
			toast.success(`Deployment queued for ${project.name}`);
			await loadProject(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to deploy project');
		} finally {
			pendingAction = null;
		}
	}
</script>

<div class="mx-auto max-w-7xl px-4 py-6 sm:px-6">
	<a href="/projects" class="mb-4 inline-flex items-center gap-1.5 text-sm font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">
		<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
			<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
		</svg>
		Projects
	</a>

	{#if loading}
		<div class="surface p-5">
			<div class="h-5 w-48 animate-pulse rounded bg-gray-200 dark:bg-gray-800"></div>
			<div class="mt-3 h-3 w-72 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
			<div class="mt-6 grid gap-3 sm:grid-cols-4">
				{#each [1, 2, 3, 4] as _}
					<div class="h-12 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>
				{/each}
			</div>
		</div>
	{:else if error || !project}
		<div class="surface border-red-200 p-6 text-sm text-red-600 dark:border-red-900/50 dark:text-red-300">
			{error || 'Project not found'}
		</div>
	{:else}
		<section class="surface overflow-hidden">
			<div class="flex flex-col gap-4 px-5 py-5 lg:flex-row lg:items-start lg:justify-between">
				<div class="min-w-0">
					<div class="flex flex-wrap items-center gap-3">
						<h1 class="truncate text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{project.name}</h1>
						<StatusBadge status={project.status} pulse />
					</div>
					<a
						href={publicProjectURL}
						target="_blank"
						rel="noopener"
						class="mt-1 inline-flex max-w-full items-center gap-1.5 truncate font-mono text-xs text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white"
					>
						{publicProjectHost}
						<svg class="h-3.5 w-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
						</svg>
					</a>
				</div>

				<div class="flex shrink-0 flex-wrap gap-2">
					<ActionButton
						on:click={handleStop}
						disabled={project.status === 'stopped' || (pendingAction !== null && pendingAction !== 'stop')}
						loading={pendingAction === 'stop'}
						loadingLabel="Stopping..."
					>
						Stop
					</ActionButton>
					<ActionButton
						on:click={handleRestart}
						disabled={pendingAction !== null && pendingAction !== 'restart'}
						loading={pendingAction === 'restart'}
						loadingLabel="Restarting..."
					>
						Restart
					</ActionButton>
					<ActionButton
						variant="primary"
						on:click={handleDeploy}
						disabled={project.status === 'building' || (pendingAction !== null && pendingAction !== 'deploy')}
						loading={pendingAction === 'deploy'}
						loadingLabel="Queueing..."
					>
						Deploy
					</ActionButton>
				</div>
			</div>

			<div class="grid border-y border-gray-100 bg-gray-50/60 dark:border-gray-800 dark:bg-gray-900/60 sm:grid-cols-2 lg:grid-cols-5">
				<div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 sm:border-r lg:border-b-0">
					<p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Repository</p>
					<p class="mt-1 truncate font-mono text-xs text-gray-800 dark:text-gray-200">{project.repoUrl}</p>
				</div>
				<div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 lg:border-b-0 lg:border-r">
					<p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Branch</p>
					<p class="mt-1 font-mono text-xs text-gray-800 dark:text-gray-200">{project.branch}</p>
				</div>
				<div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 sm:border-r lg:border-b-0">
					<p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Mode</p>
					<p class="mt-1 font-mono text-xs text-gray-800 dark:text-gray-200">{project.deployMode}{project.mainService ? ` / ${project.mainService}` : ''}</p>
				</div>
				<div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 lg:border-b-0 lg:border-r">
					<p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Port</p>
					<p class="mt-1 font-mono text-xs text-gray-800 dark:text-gray-200">{project.appPort} → {project.allocatedPort ?? 'pending'}</p>
				</div>
				<div class="px-5 py-3">
					<p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Limits</p>
					<p class="mt-1 font-mono text-xs text-gray-800 dark:text-gray-200">{project.memoryLimitMb}MB / {project.cpuLimit} CPU</p>
				</div>
			</div>

			<nav class="flex gap-1 overflow-x-auto px-3 py-2" data-sveltekit-preload-data="hover">
				{#each tabs as tab}
					<a
						href="{base}{tab.href}"
						aria-current={activeHref === tab.href ? 'page' : undefined}
						class="whitespace-nowrap rounded-md px-3 py-1.5 text-sm font-medium transition-colors
							{activeHref === tab.href
								? 'bg-gray-950 text-white dark:bg-white dark:text-gray-950'
								: 'text-gray-500 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white'}"
					>
						{tab.label}
					</a>
				{/each}
			</nav>
		</section>

		<div class="py-5">
			<slot />
		</div>
	{/if}
</div>
