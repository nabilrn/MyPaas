<script lang="ts">
	import { onMount } from 'svelte';
	import { Activity, ArrowRight, Database, History, KeyRound, Settings2 } from '@lucide/svelte';
	import { page } from '$app/stores';
	import ActionLink from '$components/ActionLink.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import StatusBadge from '$components/StatusBadge.svelte';
	import { api } from '$api';
	import { runtimeServiceSummary, selectPrimaryProjectMetric } from '$lib/utils/project-dashboard';
	import type { DBStudioStatus, Deployment, MetricsSnapshot, Project } from '$types';

	let project: Project | null = null;
	let deployments: Deployment[] = [];
	let metrics: MetricsSnapshot | null = null;
	let envCount: number | null = null;
	let dbStatus: DBStudioStatus | null = null;
	let supportingSummaryLoaded = false;
	let loading = true;
	let overviewInFlight = false;
	let metricsInFlight = false;
	let error = '';

	$: base = `/projects/${$page.params.id}`;
	$: lastDeploy = deployments.find((deployment) => deployment.id === project?.activeDeploymentId) ?? deployments[0];
	$: primaryMetric = selectPrimaryProjectMetric(metrics, project?.mainService);
	$: runtimeSummary = runtimeServiceSummary(metrics, primaryMetric);
	$: metricsUpdatedLabel = metrics?.collectedAt
		? `Updated ${new Date(metrics.collectedAt).toLocaleTimeString()}`
		: 'Waiting for runtime sample';
	$: applicationState = applicationStateFor(project, lastDeploy);
	$: runtimeLabel = project
		? project.deployMode === 'compose'
			? 'Docker Compose'
			: project.deployMode === 'dockerfile'
				? 'Dockerfile'
				: project.deployMode === 'static'
					? 'Static site'
					: 'Container image'
		: '-';
	$: deploymentSummary = project
		? [runtimeLabel, project.mainService || '', project.deployMode === 'static' ? '' : `:${project.appPort}`].filter(Boolean).join(' · ')
		: '-';
	$: databaseLabel = dbStatus?.configured
		? `${dbStatus.connection?.driver?.toUpperCase() ?? 'Database'} · ${dbStatus.connected ? 'Connected' : 'Unavailable'}`
		: supportingSummaryLoaded
			? 'Not configured'
			: 'Checking…';

	onMount(() => {
		void loadOverview();
		void loadMetricsSnapshot();
		void loadSupportingSummary();

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
		if (!background && !project) loading = true;
		try {
			const [projectResult, deploymentRows] = await Promise.all([
				api.projects.get($page.params.id ?? ''),
				api.deployments.list($page.params.id ?? '')
			]);
			project = projectResult;
			deployments = deploymentRows;
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load project dashboard';
		} finally {
			loading = false;
			overviewInFlight = false;
		}
	}

	async function loadMetricsSnapshot() {
		if (metricsInFlight) return;
		metricsInFlight = true;
		try {
			metrics = await api.metrics.snapshot($page.params.id ?? '');
		} catch {
			metrics = null;
		} finally {
			metricsInFlight = false;
		}
	}

	async function loadSupportingSummary() {
		const projectId = $page.params.id ?? '';
		const [envResult, databaseResult] = await Promise.allSettled([
			api.env.list(projectId),
			api.dbStudio.status(projectId)
		]);

		envCount = envResult.status === 'fulfilled' ? envResult.value.length : null;
		dbStatus = databaseResult.status === 'fulfilled' ? databaseResult.value : null;
		supportingSummaryLoaded = true;
	}

	function applicationStateFor(currentProject: Project | null, currentDeployment: Deployment | undefined) {
		if (!currentProject) {
			return { title: 'Loading application', body: 'Reading current project state.', action: null as null | { label: string; href: string } };
		}
		if (currentProject.status === 'building') {
			return {
				title: 'Deployment in progress',
				body: currentDeployment ? `MyPaas is processing the latest ${titleCase(currentDeployment.status)} stage.` : 'MyPaas is building the current deployment.',
				action: { label: 'View deployment', href: `${base}/deployments${currentDeployment ? `?focus=${encodeURIComponent(currentDeployment.id)}` : ''}` }
			};
		}
		if (currentProject.status === 'crashed') {
			return {
				title: 'Application crashed',
				body: 'The runtime exited unexpectedly. Check logs before restarting it.',
				action: { label: 'View logs', href: `${base}/logs` }
			};
		}
		if (currentProject.status === 'stopped') {
			return {
				title: 'Application is stopped',
				body: 'No runtime is currently serving traffic for this project.',
				action: { label: 'View deployments', href: `${base}/deployments` }
			};
		}
		return {
			title: currentDeployment ? 'Application is waiting' : 'Project ready to deploy',
			body: currentDeployment ? 'The project exists but no runtime is currently serving traffic.' : 'Source and configuration are saved. Use Deploy above to publish the first version.',
			action: currentDeployment ? { label: 'View deployments', href: `${base}/deployments` } : null
		};
	}

	function formatDuration(start: string, end: string | null): string {
		if (!end) return 'In progress';
		const seconds = Math.max(0, Math.floor((new Date(end).getTime() - new Date(start).getTime()) / 1000));
		return seconds < 60 ? `${seconds}s` : `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
	}

	function formatDate(value: string | null) {
		if (!value) return '-';
		return new Date(value).toLocaleString();
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
		<div class="surface h-56 animate-pulse"></div>
		<div class="grid gap-4 lg:grid-cols-2">
			<div class="surface h-64 animate-pulse"></div>
			<div class="surface h-64 animate-pulse"></div>
		</div>
	</div>
{:else if error && !project}
	<div class="surface overflow-hidden">
		<ErrorState title="Could not load project dashboard" message={error} on:retry={() => void loadOverview()} />
	</div>
{:else if project}
	<div class="space-y-4">
		{#if error}
			<div class="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-200">
				<p class="font-medium">Dashboard refresh failed</p>
				<p class="mt-1">{error}</p>
			</div>
		{/if}

		{#if project.status === 'building' || project.status === 'crashed' || project.status === 'pending'}
			<section class="surface overflow-hidden">
				<div class="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-2">
							<StatusBadge status={project.status} pulse />
							<h2 class="text-base font-semibold text-gray-950 dark:text-white">{applicationState.title}</h2>
						</div>
						<p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{applicationState.body}</p>
					</div>
					{#if applicationState.action}
						<ActionLink href={applicationState.action.href} variant="secondary" size="xs">
							<ArrowRight slot="icon" class="h-3.5 w-3.5" />
							{applicationState.action.label}
						</ActionLink>
					{/if}
				</div>
			</section>
		{/if}

		{#if project.deployMode !== 'static'}
			<section class="surface overflow-hidden">
				<div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-neutral-800">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Runtime summary</h2>
						<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{runtimeSummary.label} · {metricsUpdatedLabel}</p>
					</div>
					<ActionLink href={`${base}/metrics`} variant="ghost" size="xs">
						<Activity slot="icon" class="h-3.5 w-3.5" />
						Diagnostics
					</ActionLink>
				</div>
				{#if primaryMetric}
					<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 sm:grid-cols-3 sm:divide-x sm:divide-y-0">
						<div class="p-5">
							<p class="metric-label">CPU</p>
							<p class="metric-value mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primaryMetric.cpu.toFixed(2)}%</p>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Current runtime sample</p>
						</div>
						<div class="p-5">
							<p class="metric-label">Memory</p>
							<p class="metric-value mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primaryMetric.memoryMb.toFixed(1)} <span class="text-sm font-medium text-gray-500 dark:text-gray-400">MB</span></p>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{primaryMetric.memoryLimitMb.toFixed(0)} MB limit</p>
						</div>
						<div class="p-5">
							<p class="metric-label">Uptime</p>
							<p class="metric-value mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primaryMetric.uptime}</p>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{primaryMetric.service}</p>
						</div>
					</div>
				{:else}
					<div class="px-5 py-8">
						<EmptyState title="No active runtime metrics." description="CPU and memory samples are reported only while a container runtime is active." compact />
					</div>
				{/if}
			</section>
		{/if}

		<div class="grid gap-4 lg:grid-cols-[minmax(0,1.25fr)_minmax(20rem,0.75fr)]">
			<section class="surface overflow-hidden">
				<div class="flex items-start justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-neutral-800">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Latest deployment</h2>
						<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">The newest deployment attempt for this project.</p>
					</div>
					<ActionLink href={`${base}/deployments`} variant="ghost" size="xs">
						<History slot="icon" class="h-3.5 w-3.5" />
						View all
					</ActionLink>
				</div>

				{#if lastDeploy}
					<div class="p-5">
						<div class="flex flex-wrap items-center justify-between gap-3">
							<div class="flex min-w-0 items-center gap-3">
								<StatusBadge status={lastDeploy.status} />
								<p class="truncate font-mono text-sm font-medium text-gray-950 dark:text-white">
									{project.sourceType === 'registry' ? (lastDeploy.imageTag ?? project.imageRef ?? '-') : (lastDeploy.commitSha?.slice(0, 8) ?? '-')}
								</p>
							</div>
							<p class="metric-value text-xs text-gray-500 dark:text-gray-400">{formatDuration(lastDeploy.startedAt, lastDeploy.finishedAt)}</p>
						</div>
						<p class="mt-4 text-sm text-gray-800 dark:text-gray-200">{lastDeploy.commitMessage || 'No deployment message'}</p>
						<div class="mt-4 flex flex-wrap gap-x-6 gap-y-2 text-xs text-gray-500 dark:text-gray-400">
							<span>Triggered by <span class="capitalize text-gray-700 dark:text-gray-300">{lastDeploy.triggeredBy}</span></span>
							<span>{formatDate(lastDeploy.startedAt)}</span>
						</div>
					</div>
				{:else}
					<div class="px-5 py-8">
						<EmptyState title="No deployment yet." description="Use Deploy above when you are ready to publish the first version." compact />
					</div>
				{/if}
			</section>

			<section class="surface overflow-hidden">
				<div class="border-b border-gray-100 px-5 py-4 dark:border-neutral-800">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Project essentials</h2>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Open configuration only when you need to change it.</p>
				</div>

				<div class="divide-y divide-gray-100 dark:divide-neutral-800">
					<a href={`${base}/env`} class="group flex items-center justify-between gap-4 px-5 py-4 hover:bg-gray-50/70 dark:hover:bg-neutral-900/60">
						<div class="flex min-w-0 items-center gap-3">
							<KeyRound class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0">
								<p class="text-sm font-medium text-gray-950 dark:text-white">Environment</p>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{envCount === null ? (supportingSummaryLoaded ? 'Unavailable' : 'Checking…') : `${envCount} variable${envCount === 1 ? '' : 's'} configured`}</p>
							</div>
						</div>
						<ArrowRight class="h-4 w-4 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
					</a>

					<a href={`${base}/database`} class="group flex items-center justify-between gap-4 px-5 py-4 hover:bg-gray-50/70 dark:hover:bg-neutral-900/60">
						<div class="flex min-w-0 items-center gap-3">
							<Database class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0">
								<p class="text-sm font-medium text-gray-950 dark:text-white">Database</p>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{databaseLabel}</p>
							</div>
						</div>
						<ArrowRight class="h-4 w-4 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
					</a>

					<a href={`${base}/settings`} class="group flex items-center justify-between gap-4 px-5 py-4 hover:bg-gray-50/70 dark:hover:bg-neutral-900/60">
						<div class="flex min-w-0 items-center gap-3">
							<Settings2 class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0">
								<p class="text-sm font-medium text-gray-950 dark:text-white">Deployment setup</p>
								<p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{deploymentSummary}</p>
								<p class="metric-value mt-1 text-[11px] text-gray-400 dark:text-gray-500">{project.memoryLimitMb} MB · {project.cpuLimit} CPU</p>
							</div>
						</div>
						<ArrowRight class="h-4 w-4 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
					</a>
				</div>
			</section>
		</div>
	</div>
{/if}
