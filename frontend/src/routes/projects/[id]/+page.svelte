<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import StatusBadge from '$components/StatusBadge.svelte';
	import { api } from '$api';
	import type { ContainerMetrics, Deployment, MetricsSnapshot, Project } from '$types';

	let project: Project | null = null;
	let deployments: Deployment[] = [];
	let metrics: MetricsSnapshot | null = null;
	let loading = true;
	let error = '';

	$: lastDeploy = deployments.find((d) => d.id === project?.activeDeploymentId) ?? deployments[0];
	$: primaryMetric = metrics?.items[0] ?? null;
	$: memoryPercent = primaryMetric && primaryMetric.memoryLimitMb > 0
		? Math.min((primaryMetric.memoryMb / primaryMetric.memoryLimitMb) * 100, 100)
		: 0;
	$: configRows = project
		? [
				['Repository', project.repoUrl],
				['Branch', project.branch],
				['Deploy mode', project.deployMode],
				['App port', String(project.appPort)],
				['Memory', `${project.memoryLimitMb} MB`],
				['CPU', `${project.cpuLimit} cores`]
			]
		: [];

	onMount(() => {
		void load();
		const id = setInterval(load, 3000);
		return () => clearInterval(id);
	});

	async function load() {
		try {
			const [projectResult, deploymentRows, metricSnapshot] = await Promise.all([
				api.projects.get($page.params.id),
				api.deployments.list($page.params.id),
				loadMetrics($page.params.id)
			]);
			project = projectResult;
			deployments = deploymentRows;
			metrics = metricSnapshot;
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load project overview';
		} finally {
			loading = false;
		}
	}

	async function loadMetrics(projectId: string): Promise<MetricsSnapshot | null> {
		try {
			return await api.metrics.snapshot(projectId);
		} catch {
			return null;
		}
	}

	function metricMemory(metric: ContainerMetrics | null) {
		if (!metric) return '-';
		return `${metric.memoryMb.toFixed(1)}`;
	}

	function formatDate(value: string | null) {
		if (!value) return '-';
		return new Date(value).toLocaleString();
	}
</script>

<svelte:head>
	<title>{project?.name ?? 'Project'} · MyPaas</title>
</svelte:head>

{#if loading || !project}
	<div class="space-y-4">
		<div class="surface grid gap-0 overflow-hidden sm:grid-cols-4">
			{#each [1, 2, 3, 4] as _}
				<div class="border-b border-gray-100 p-5 dark:border-gray-800 sm:border-b-0 sm:border-r">
					<div class="h-3 w-20 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
					<div class="mt-3 h-7 w-24 animate-pulse rounded bg-gray-200 dark:bg-gray-800"></div>
				</div>
			{/each}
		</div>
		<div class="grid gap-4 lg:grid-cols-2">
			<div class="surface h-56 animate-pulse"></div>
			<div class="surface h-56 animate-pulse"></div>
		</div>
	</div>
{:else}
	<div class="space-y-4">
		{#if error}
			<div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-200">
				{error}
			</div>
		{/if}

		<section class="surface overflow-hidden">
			<div class="grid divide-y divide-gray-100 dark:divide-gray-800 sm:grid-cols-2 sm:divide-x sm:divide-y-0 lg:grid-cols-4">
				<div class="p-5">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Status</p>
					<div class="mt-3"><StatusBadge status={project.status} pulse /></div>
				</div>
				<div class="p-5">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Uptime</p>
					<p class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primaryMetric?.uptime ?? '-'}</p>
				</div>
				<div class="p-5">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Memory</p>
					<p class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">
						{metricMemory(primaryMetric)}
						<span class="text-sm font-normal text-gray-500">/{primaryMetric?.memoryLimitMb.toFixed(0) ?? project.memoryLimitMb} MB</span>
					</p>
					<div class="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-full rounded-full bg-emerald-500" style="width: {memoryPercent}%"></div>
					</div>
				</div>
				<div class="p-5">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">CPU</p>
					<p class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">
						{primaryMetric ? primaryMetric.cpu.toFixed(2) : '-'}<span class="text-sm font-normal text-gray-500">%</span>
					</p>
					<div class="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-full rounded-full bg-sky-500" style="width: {primaryMetric ? Math.min(primaryMetric.cpu, 100) : 0}%"></div>
					</div>
				</div>
			</div>
		</section>

		<div class="grid gap-4 lg:grid-cols-2">
			<section class="surface overflow-hidden">
				<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Configuration</h2>
				</div>
				<dl class="divide-y divide-gray-100 dark:divide-gray-800">
					{#each configRows as [k, v]}
						<div class="grid grid-cols-[8rem_minmax(0,1fr)] gap-4 px-5 py-3 text-sm">
							<dt class="text-gray-500 dark:text-gray-400">{k}</dt>
							<dd class="truncate font-medium text-gray-950 dark:text-white">{v}</dd>
						</div>
					{/each}
				</dl>
			</section>

			<section class="surface overflow-hidden">
				<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Last deployment</h2>
				</div>
				{#if lastDeploy}
					<div class="divide-y divide-gray-100 dark:divide-gray-800">
						<div class="flex items-center justify-between gap-4 px-5 py-3 text-sm">
							<span class="text-gray-500 dark:text-gray-400">Commit</span>
							<span class="font-mono font-medium text-gray-950 dark:text-white">{lastDeploy.commitSha?.slice(0, 8) ?? '-'}</span>
						</div>
						<div class="flex items-center justify-between gap-4 px-5 py-3 text-sm">
							<span class="text-gray-500 dark:text-gray-400">Status</span>
							<StatusBadge status={lastDeploy.status} />
						</div>
						<div class="flex items-center justify-between gap-4 px-5 py-3 text-sm">
							<span class="text-gray-500 dark:text-gray-400">Triggered by</span>
							<span class="capitalize text-gray-950 dark:text-white">{lastDeploy.triggeredBy}</span>
						</div>
						<div class="grid grid-cols-[8rem_minmax(0,1fr)] gap-4 px-5 py-3 text-sm">
							<span class="text-gray-500 dark:text-gray-400">Message</span>
							<span class="truncate text-gray-950 dark:text-white">{lastDeploy.commitMessage || '-'}</span>
						</div>
						<div class="flex items-center justify-between gap-4 px-5 py-3 text-sm">
							<span class="text-gray-500 dark:text-gray-400">Started</span>
							<span class="text-gray-950 dark:text-white">{formatDate(lastDeploy.startedAt)}</span>
						</div>
					</div>
				{:else}
					<p class="p-5 text-sm text-gray-500 dark:text-gray-400">No deployment yet.</p>
				{/if}
			</section>
		</div>
	</div>
{/if}
