<script lang="ts">
	import { onMount } from 'svelte';
	import StatusBadge from '$components/StatusBadge.svelte';
	import EmptyState  from '$components/EmptyState.svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { Project, QuotaUsage } from '$types';

	let projects: Project[] = [];
	let quota: QuotaUsage | null = null;
	let loading = true;
	let error = '';
	let deployingProjectId = '';

	$: memoryConfiguredPercent = quota && quota.memoryLimitMb > 0 ? Math.min(100, (quota.memoryUsedMb / quota.memoryLimitMb) * 100) : 0;
	$: memoryRuntimePercent = quota && quota.memoryUsedMb > 0 ? Math.min(100, (quota.memoryRuntimeMb / quota.memoryUsedMb) * 100) : 0;
	$: cpuConfiguredPercent = quota && quota.cpuLimit > 0 ? Math.min(100, (quota.cpuUsed / quota.cpuLimit) * 100) : 0;
	$: projectPercent = quota && quota.projectLimit > 0 ? Math.min(100, (quota.projectCount / quota.projectLimit) * 100) : 0;
	$: quotaWarning = quota && (memoryConfiguredPercent >= 80 || cpuConfiguredPercent >= 80 || projectPercent >= 80);

	onMount(loadProjects);

	async function loadProjects(background = false) {
		if (!background) {
			loading = true;
		}
		error = '';
		try {
			[projects, quota] = await Promise.all([api.projects.list(), api.me.quota()]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load projects';
		} finally {
			if (!background) {
				loading = false;
			}
		}
	}

	async function handleDeploy(id: string, name: string) {
		if (deployingProjectId) return;
		deployingProjectId = id;
		try {
			await api.projects.deploy(id);
			toast.success(`Deployment queued for ${name}`);
			await loadProjects(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to trigger deployment');
		} finally {
			deployingProjectId = '';
		}
	}

	function formatDate(value: string) {
		return new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
	}
</script>

<svelte:head>
	<title>Projects · MyPaas</title>
</svelte:head>

<div class="mx-auto max-w-7xl px-4 py-7 sm:px-6">
	<header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
		<div>
			<p class="text-xs font-medium uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400">Control plane</p>
			<h1 class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">Projects</h1>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
				{projects.length} project{projects.length !== 1 ? 's' : ''} connected to MyPaas.
			</p>
		</div>
		<a
			href="/projects/new"
			class="inline-flex min-h-10 items-center justify-center gap-2 rounded-md bg-gray-950 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-gray-950 focus:ring-offset-2 dark:bg-white dark:text-gray-950 dark:hover:bg-gray-200 dark:focus:ring-white dark:focus:ring-offset-gray-950"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
			</svg>
			New project
		</a>
	</header>

	{#if quota}
		<section class="surface mb-5 overflow-hidden">
			<div class="grid divide-y divide-gray-100 dark:divide-gray-800 sm:grid-cols-3 sm:divide-x sm:divide-y-0">
				<div class="p-4">
					<div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
						<span>Configured memory</span>
						<span>{quota.memoryUsedMb}/{quota.memoryLimitMb} MB</span>
					</div>
					<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-full rounded-full bg-gray-950 dark:bg-white" style="width: {memoryConfiguredPercent}%"></div>
					</div>
					<div class="mt-3 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
						<span>Runtime memory</span>
						<span>{quota.memoryRuntimeMb} MB live</span>
					</div>
					<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-full rounded-full bg-emerald-500" style="width: {memoryRuntimePercent}%"></div>
					</div>
				</div>
				<div class="p-4">
					<div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
						<span>Configured CPU</span>
						<span>{quota.cpuUsed.toFixed(2)}/{quota.cpuLimit.toFixed(2)}</span>
					</div>
					<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-full rounded-full bg-sky-500" style="width: {cpuConfiguredPercent}%"></div>
					</div>
					<div class="mt-3 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
						<span>Runtime CPU</span>
						<span>{quota.cpuRuntime.toFixed(1)}%</span>
					</div>
				</div>
				<div class="p-4">
					<div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
						<span>Projects</span>
						<span>{quota.projectCount}/{quota.projectLimit}</span>
					</div>
					<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-full rounded-full bg-violet-500" style="width: {projectPercent}%"></div>
					</div>
				</div>
			</div>
			{#if quotaWarning}
				<div class="border-t border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-200">
					Resource usage is approaching the configured limit.
				</div>
			{/if}
		</section>
	{/if}

	<section class="surface overflow-hidden">
		<div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-gray-800">
			<div>
				<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Deployments</h2>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Live project inventory and quick deploy actions.</p>
			</div>
			<button on:click={() => loadProjects()} class="text-xs font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">
				Refresh
			</button>
		</div>

		{#if loading}
			<div class="divide-y divide-gray-100 dark:divide-gray-800">
				{#each [1, 2, 3] as _}
					<div class="grid gap-4 px-4 py-4 lg:grid-cols-[minmax(0,1fr)_9rem_9rem_8rem_8rem]">
						<div class="space-y-2">
							<div class="h-4 w-40 animate-pulse rounded bg-gray-200 dark:bg-gray-800"></div>
							<div class="h-3 w-64 animate-pulse rounded bg-gray-100 dark:bg-gray-800/70"></div>
						</div>
						<div class="h-6 w-20 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
						<div class="h-4 w-24 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
						<div class="h-4 w-20 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
						<div class="h-8 w-24 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
					</div>
				{/each}
			</div>
		{:else if error}
			<div class="p-5 text-sm text-red-600 dark:text-red-300">
				{error}
				<button on:click={() => loadProjects()} class="ml-3 font-medium underline">Retry</button>
			</div>
		{:else if projects.length === 0}
			<EmptyState
				title="No projects yet"
				description="Connect a Git repository and MyPaas will build it from Dockerfile or Compose."
				actionLabel="New project"
				actionHref="/projects/new"
			/>
		{:else}
			<div class="hidden grid-cols-[minmax(0,1fr)_9rem_9rem_8rem_8rem] border-b border-gray-100 bg-gray-50/70 px-4 py-2 text-xs font-medium uppercase tracking-wide text-gray-500 dark:border-gray-800 dark:bg-gray-900/70 dark:text-gray-400 lg:grid">
				<span>Project</span>
				<span>Status</span>
				<span>Runtime</span>
				<span>Updated</span>
				<span class="text-right">Actions</span>
			</div>
			<div class="divide-y divide-gray-100 dark:divide-gray-800">
				{#each projects as project}
					<div class="grid gap-3 px-4 py-4 transition-colors hover:bg-gray-50/80 dark:hover:bg-gray-900/70 lg:grid-cols-[minmax(0,1fr)_9rem_9rem_8rem_8rem] lg:items-center">
						<div class="min-w-0">
							<div class="flex items-center gap-3">
								<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-gray-200 bg-gray-50 font-mono text-xs font-semibold uppercase text-gray-500 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400">
									{project.name.slice(0, 2)}
								</span>
								<div class="min-w-0">
									<a href="/projects/{project.id}" class="block truncate text-sm font-semibold text-gray-950 hover:underline dark:text-white">
										{project.name}
									</a>
									<p class="mt-0.5 truncate font-mono text-xs text-gray-500 dark:text-gray-400">
										{project.subdomain}
									</p>
								</div>
							</div>
						</div>
						<div><StatusBadge status={project.status} pulse /></div>
						<div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
							<span class="rounded border border-gray-200 px-1.5 py-0.5 font-mono dark:border-gray-800">{project.deployMode}</span>
							<span>{project.memoryLimitMb}MB</span>
						</div>
						<div class="text-xs text-gray-500 dark:text-gray-400">
							{formatDate(project.updatedAt)}
						</div>
						<div class="flex items-center justify-start gap-2 lg:justify-end">
							<a
								href="/projects/{project.id}"
								class="inline-flex min-h-8 items-center justify-center rounded-md border border-gray-300 bg-white px-2.5 py-1.5 text-xs font-medium text-gray-800 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-900"
							>
								Open
							</a>
							{#if project.status !== 'building'}
								<ActionButton
									variant="primary"
									size="xs"
									on:click={() => handleDeploy(project.id, project.name)}
									loading={deployingProjectId === project.id}
									loadingLabel="Queueing..."
									disabled={deployingProjectId !== '' && deployingProjectId !== project.id}
								>
									Deploy
								</ActionButton>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</section>
</div>
