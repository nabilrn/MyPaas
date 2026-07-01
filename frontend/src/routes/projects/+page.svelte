<script lang="ts">
	import { onMount } from 'svelte';
	import StatusBadge from '$components/StatusBadge.svelte';
	import EmptyState  from '$components/EmptyState.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { Project, QuotaUsage } from '$types';

	let projects: Project[] = [];
	let quota: QuotaUsage | null = null;
	let loading = true;
	let error = '';

	$: memoryPercent = quota && quota.memoryLimitMb > 0 ? Math.min(100, (quota.memoryUsedMb / quota.memoryLimitMb) * 100) : 0;
	$: cpuPercent = quota && quota.cpuLimit > 0 ? Math.min(100, (quota.cpuUsed / quota.cpuLimit) * 100) : 0;
	$: projectPercent = quota && quota.projectLimit > 0 ? Math.min(100, (quota.projectCount / quota.projectLimit) * 100) : 0;
	$: quotaWarning = quota && (memoryPercent >= 80 || cpuPercent >= 80 || projectPercent >= 80);

	onMount(loadProjects);

	async function loadProjects() {
		loading = true;
		error = '';
		try {
			[projects, quota] = await Promise.all([api.projects.list(), api.me.quota()]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load projects';
		} finally {
			loading = false;
		}
	}

	async function handleDeploy(id: string, name: string) {
		try {
			await api.projects.deploy(id);
			toast.success(`Deployment queued for ${name}`);
			await loadProjects();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to trigger deployment');
		}
	}
</script>

<svelte:head>
	<title>Projects · MyPaas</title>
</svelte:head>

<div class="mx-auto max-w-7xl px-4 py-8 sm:px-6">
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h1 class="text-xl font-bold text-gray-900 dark:text-white">Projects</h1>
			<p class="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
				{projects.length} project{projects.length !== 1 ? 's' : ''}
			</p>
		</div>
		<a
			href="/projects/new"
			class="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium
				   text-white hover:bg-brand-700 focus:outline-none focus:ring-2 focus:ring-brand-500"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
			</svg>
			New project
		</a>
	</div>

	{#if quota}
		<div class="mb-6 rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900">
			<div class="mb-3 flex items-center justify-between gap-3">
				<h2 class="text-sm font-semibold text-gray-900 dark:text-white">Quota</h2>
				{#if quotaWarning}
					<span class="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-950/40 dark:text-amber-300">
						Near limit
					</span>
				{/if}
			</div>
			<div class="grid gap-4 sm:grid-cols-3">
				<div>
					<div class="mb-1 flex justify-between text-xs text-gray-500 dark:text-gray-400">
						<span>Memory</span>
						<span>{quota.memoryUsedMb}/{quota.memoryLimitMb} MB</span>
					</div>
					<div class="h-1.5 rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-1.5 rounded-full bg-brand-600" style="width: {memoryPercent}%"></div>
					</div>
				</div>
				<div>
					<div class="mb-1 flex justify-between text-xs text-gray-500 dark:text-gray-400">
						<span>CPU</span>
						<span>{quota.cpuUsed.toFixed(2)}/{quota.cpuLimit.toFixed(2)}</span>
					</div>
					<div class="h-1.5 rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-1.5 rounded-full bg-emerald-600" style="width: {cpuPercent}%"></div>
					</div>
				</div>
				<div>
					<div class="mb-1 flex justify-between text-xs text-gray-500 dark:text-gray-400">
						<span>Projects</span>
						<span>{quota.projectCount}/{quota.projectLimit}</span>
					</div>
					<div class="h-1.5 rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-1.5 rounded-full bg-sky-600" style="width: {projectPercent}%"></div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- Empty state -->
	{#if loading}
		<div class="rounded-xl border border-gray-200 bg-white p-6 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400">
			Loading projects...
		</div>
	{:else if error}
		<div class="rounded-xl border border-red-200 bg-white p-6 text-sm text-red-600 dark:border-red-900/50 dark:bg-gray-900 dark:text-red-400">
			{error}
			<button on:click={loadProjects} class="ml-3 font-medium underline">Retry</button>
		</div>
	{:else if projects.length === 0}
		<EmptyState
			title="No projects yet"
			description="Connect a git repository to get started."
			actionLabel="New project"
			actionHref="/projects/new"
		/>
	{:else}
		<!-- Project grid -->
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each projects as project}
				<div class="group rounded-xl border border-gray-200 bg-white p-5 transition-shadow hover:shadow-md
							dark:border-gray-800 dark:bg-gray-900">
					<div class="mb-3 flex items-start justify-between">
						<div class="min-w-0">
							<a
								href="/projects/{project.id}"
								class="block truncate font-semibold text-gray-900 hover:text-brand-600
									   dark:text-white dark:hover:text-brand-400"
							>
								{project.name}
							</a>
							<p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">
								{project.subdomain}
							</p>
						</div>
						<StatusBadge status={project.status} pulse />
					</div>

					<!-- Meta -->
					<div class="mb-4 flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
						<span class="flex items-center gap-1">
							<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14M12 5l7 7-7 7" />
							</svg>
							{project.branch}
						</span>
						<span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono dark:bg-gray-800">
							{project.deployMode}
						</span>
						<span>{project.memoryLimitMb}MB</span>
					</div>

					<!-- Actions -->
					<div class="flex items-center gap-2">
						<a
							href="/projects/{project.id}"
							class="flex-1 rounded-md border border-gray-200 px-3 py-1.5 text-center text-xs font-medium
								   text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
						>
							View
						</a>
						{#if project.status !== 'building'}
							<button
								on:click={() => handleDeploy(project.id, project.name)}
								class="flex-1 rounded-md bg-brand-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-brand-700"
							>
								Deploy
							</button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
