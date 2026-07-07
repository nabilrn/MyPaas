<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import ResourceMeter from '$components/ResourceMeter.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import StatTile from '$components/StatTile.svelte';
	import StatusBadge from '$components/StatusBadge.svelte';
	import { api } from '$api';
	import type { ContainerMetrics, Deployment, MetricsSnapshot, Project } from '$types';

	type Tone = 'neutral' | 'success' | 'info' | 'warning' | 'danger';

	let project: Project | null = null;
	let deployments: Deployment[] = [];
	let metrics: MetricsSnapshot | null = null;
	let loading = true;
	let overviewInFlight = false;
	let metricsInFlight = false;
	let error = '';

	$: lastDeploy = deployments.find((d) => d.id === project?.activeDeploymentId) ?? deployments[0];
	$: primaryMetric = metrics?.items[0] ?? null;
	$: memoryPercent = primaryMetric && primaryMetric.memoryLimitMb > 0
		? Math.min((primaryMetric.memoryMb / primaryMetric.memoryLimitMb) * 100, 100)
		: 0;
	$: cpuPercent = primaryMetric ? Math.min(primaryMetric.cpu, 100) : 0;
	$: statusTone = (project?.status === 'running'
		? 'success'
		: project?.status === 'building'
			? 'warning'
			: project?.status === 'crashed'
				? 'danger'
				: project?.status === 'pending'
					? 'info'
					: 'neutral') as Tone;
	$: lastDeployCommit = lastDeploy?.commitSha?.slice(0, 8) ?? '-';
	$: lastDeployDuration = lastDeploy ? formatDuration(lastDeploy.startedAt, lastDeploy.finishedAt) : '-';
	$: metricsUpdatedLabel = metrics?.collectedAt
		? `Updated ${new Date(metrics.collectedAt).toLocaleTimeString()}`
		: 'Waiting for metrics';
	$: configRows = project
		? [
				['Repository', project.repoUrl],
				['Branch', project.branch],
				['Deploy mode', project.deployMode],
				['Main service', project.mainService ?? '-'],
				['App port', String(project.appPort)],
				['Allocated port', project.allocatedPort ? String(project.allocatedPort) : 'pending'],
				['Memory', `${project.memoryLimitMb} MB`],
				['CPU', `${project.cpuLimit} cores`]
			]
		: [];

	onMount(() => {
		void loadOverview();
		void loadMetricsSnapshot();
		const overviewInterval = setInterval(() => void loadOverview(true), 5000);
		const metricsInterval = setInterval(() => void loadMetricsSnapshot(), 5000);
		return () => {
			clearInterval(overviewInterval);
			clearInterval(metricsInterval);
		};
	});

	async function loadOverview(background = false) {
		if (overviewInFlight) return;
		overviewInFlight = true;
		if (!background && !project) {
			loading = true;
		}
		try {
			const [projectResult, deploymentRows] = await Promise.all([
				api.projects.get($page.params.id),
				api.deployments.list($page.params.id)
			]);
			project = projectResult;
			deployments = deploymentRows;
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load project overview';
		} finally {
			loading = false;
			overviewInFlight = false;
		}
	}

	async function loadMetricsSnapshot() {
		if (metricsInFlight) return;
		metricsInFlight = true;
		try {
			metrics = await api.metrics.snapshot($page.params.id);
		} catch {
			metrics = null;
		} finally {
			metricsInFlight = false;
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

	function formatDuration(start: string, end: string | null): string {
		if (!end) return '-';
		const seconds = Math.max(0, Math.floor((new Date(end).getTime() - new Date(start).getTime()) / 1000));
		return seconds < 60 ? `${seconds}s` : `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
	}

	function titleCase(value: string) {
		return value.replace(/_/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase());
	}
</script>

<svelte:head>
	<title>{project?.name ?? 'Project'} · MyPaas</title>
</svelte:head>

{#if loading}
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
{:else if error && !project}
	<div class="surface overflow-hidden">
		<ErrorState title="Could not load project overview" message={error} on:retry={() => void loadOverview()} />
	</div>
{:else if project}
	<div class="space-y-4">
		{#if error}
			<div class="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-200">
				<p class="font-medium">Overview refresh failed</p>
				<p class="mt-1">{error}</p>
			</div>
		{/if}

		<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
			<StatTile label="Project status" value={titleCase(project.status)} detail={metricsUpdatedLabel} tone={statusTone}>
				<StatusBadge status={project.status} pulse />
			</StatTile>
			<StatTile label="Uptime" value={primaryMetric?.uptime ?? '-'} detail={primaryMetric?.service ?? 'No live container'} tone="info" />
			<StatTile
				label="Memory"
				value={`${metricMemory(primaryMetric)}/${primaryMetric?.memoryLimitMb.toFixed(0) ?? project.memoryLimitMb} MB`}
				detail={`${memoryPercent.toFixed(0)}% configured limit`}
				tone={memoryPercent >= 90 ? 'danger' : memoryPercent >= 75 ? 'warning' : 'success'}
			>
				<ResourceMeter
					label="Memory usage"
					value={`${metricMemory(primaryMetric)} MB`}
					detail={`${primaryMetric?.memoryLimitMb.toFixed(0) ?? project.memoryLimitMb} MB limit`}
					percent={memoryPercent}
					tone={memoryPercent >= 90 ? 'danger' : memoryPercent >= 75 ? 'warning' : 'success'}
				/>
			</StatTile>
			<StatTile
				label="CPU"
				value={primaryMetric ? `${primaryMetric.cpu.toFixed(2)}%` : '-'}
				detail={primaryMetric?.service ?? 'No live sample'}
				tone={cpuPercent >= 90 ? 'danger' : cpuPercent >= 75 ? 'warning' : 'info'}
			>
				<ResourceMeter
					label="CPU usage"
					value={primaryMetric ? `${primaryMetric.cpu.toFixed(2)}%` : '-'}
					detail="Runtime sample"
					percent={cpuPercent}
					tone={cpuPercent >= 90 ? 'danger' : cpuPercent >= 75 ? 'warning' : 'info'}
				/>
			</StatTile>
		</div>

		<div class="grid gap-4 lg:grid-cols-2">
			<SectionPanel
				title="Configuration"
				description="Repository, route, runtime, and quota values used by the deploy engine."
				contentClass="p-0"
			>
				<dl class="divide-y divide-gray-100 dark:divide-gray-800">
					{#each configRows as [k, v]}
						<div class="grid grid-cols-[8rem_minmax(0,1fr)] gap-4 px-5 py-3 text-sm">
							<dt class="text-gray-500 dark:text-gray-400">{k}</dt>
							<dd class="truncate font-medium text-gray-950 dark:text-white">{v}</dd>
						</div>
					{/each}
				</dl>
			</SectionPanel>

			<SectionPanel
				title="Last deployment"
				description="Most recent deployment record for this project."
				contentClass="p-0"
			>
				{#if lastDeploy}
					<div class="divide-y divide-gray-100 dark:divide-gray-800">
						<div class="flex items-center justify-between gap-4 px-5 py-3 text-sm">
							<span class="text-gray-500 dark:text-gray-400">Commit</span>
							<span class="font-mono font-medium text-gray-950 dark:text-white">{lastDeployCommit}</span>
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
							<span class="text-gray-500 dark:text-gray-400">Duration</span>
							<span class="font-mono text-gray-950 dark:text-white">{lastDeployDuration}</span>
						</div>
						<div class="flex items-center justify-between gap-4 px-5 py-3 text-sm">
							<span class="text-gray-500 dark:text-gray-400">Started</span>
							<span class="text-gray-950 dark:text-white">{formatDate(lastDeploy.startedAt)}</span>
						</div>
					</div>
				{:else}
					<EmptyState
						title="No deployment yet."
						description="Trigger the first deploy from the project actions panel to create a deployment record."
						compact
					/>
				{/if}
			</SectionPanel>
		</div>
	</div>
{/if}
