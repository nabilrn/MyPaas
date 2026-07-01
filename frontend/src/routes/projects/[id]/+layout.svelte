<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import StatusBadge from '$components/StatusBadge.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { Project } from '$types';

	let project: Project | null = null;
	let loading = true;
	let error = '';

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

	onMount(loadProject);

	async function loadProject() {
		loading = true;
		error = '';
		try {
			project = await api.projects.get($page.params.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load project';
		} finally {
			loading = false;
		}
	}

	async function handleStop() {
		if (!project) return;
		try {
			await api.projects.stop(project.id);
			toast.success(`Stopped ${project.name}`);
			await loadProject();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to stop project');
		}
	}
	async function handleRestart() {
		if (!project) return;
		try {
			await api.projects.restart(project.id);
			toast.success(`Restarted ${project.name}`);
			await loadProject();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to restart project');
		}
	}
	async function handleDeploy() {
		if (!project) return;
		try {
			await api.projects.deploy(project.id);
			toast.success(`Deployment queued for ${project.name}`);
			await loadProject();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to deploy project');
		}
	}
</script>

<div class="mx-auto max-w-7xl px-4 pt-6 sm:px-6">
	<!-- Breadcrumb -->
	<a href="/projects" class="mb-4 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-900 dark:hover:text-white">
		<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
			<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
		</svg>
		Projects
	</a>

	{#if loading}
		<div class="rounded-xl border border-gray-200 bg-white p-6 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400">
			Loading project...
		</div>
	{:else if error || !project}
		<div class="rounded-xl border border-red-200 bg-white p-6 text-sm text-red-600 dark:border-red-900/50 dark:bg-gray-900 dark:text-red-400">
			{error || 'Project not found'}
		</div>
	{:else}
	<!-- Project header -->
	<div class="mb-4 flex flex-wrap items-start gap-4">
		<div class="min-w-0 flex-1">
			<div class="flex items-center gap-3">
				<h1 class="text-xl font-bold text-gray-900 dark:text-white">{project.name}</h1>
				<StatusBadge status={project.status} pulse />
			</div>
			<a
				href="https://{project.subdomain}.nabilrizkinavisa.me"
				target="_blank"
				rel="noopener"
				class="mt-0.5 inline-flex items-center gap-1 text-sm text-brand-600 hover:underline dark:text-brand-400"
			>
				{project.subdomain}.nabilrizkinavisa.me
				<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
				</svg>
			</a>
		</div>

		<!-- Action buttons -->
		<div class="flex shrink-0 gap-2">
			<button
				on:click={handleStop}
				disabled={project.status === 'stopped'}
				class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700
					   hover:bg-gray-50 disabled:opacity-40 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
			>
				Stop
			</button>
			<button
				on:click={handleRestart}
				class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700
					   hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
			>
				Restart
			</button>
			<button
				on:click={handleDeploy}
				disabled={project.status === 'building'}
				class="rounded-lg bg-brand-600 px-3 py-1.5 text-sm font-medium text-white
					   hover:bg-brand-700 disabled:opacity-40"
			>
				Deploy
			</button>
		</div>
	</div>

	<!-- Tabs -->
	<div class="border-b border-gray-200 dark:border-gray-800">
		<nav class="-mb-px flex gap-0 overflow-x-auto">
			{#each tabs as tab}
				<a
					href="{base}{tab.href}"
					class="whitespace-nowrap border-b-2 px-4 py-2.5 text-sm font-medium transition-colors
						   {activeHref === tab.href
							? 'border-brand-600 text-brand-600 dark:border-brand-400 dark:text-brand-400'
							: 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'}"
				>
					{tab.label}
				</a>
			{/each}
		</nav>
	</div>

	<!-- Tab content -->
	<div class="py-6">
		<slot />
	</div>
	{/if}
</div>
