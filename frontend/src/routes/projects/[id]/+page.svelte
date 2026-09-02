<script lang="ts">
	import { onMount } from 'svelte';
	import { Activity, ArrowRight, Database, ExternalLink, History, KeyRound, Settings2 } from '@lucide/svelte';
	import { page } from '$app/stores';
	import ActionLink from '$components/ActionLink.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import EnvironmentVariablesDialog from '$components/EnvironmentVariablesDialog.svelte';
	import ProjectObservability from '$components/ProjectObservability.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import StatusBadge from '$components/StatusBadge.svelte';
	import { api, type ProjectHTTPRoute } from '$api';
	import { deploymentHistoryLabel } from '$lib/utils/deploymentHistory';
	import { selectPrimaryProjectMetric } from '$lib/utils/project-dashboard';
	import { deriveProjectOperationalState } from '$lib/utils/project-operational-state';
	import { projectRouteURL } from '$lib/utils/urls';
	import { projectStreamMetrics } from '$stores/project-stream';
	import type { DBStudioStatus, Deployment, Project } from '$types';

	let project: Project | null = null;
	let deployments: Deployment[] = [];
	let envCount: number | null = null;
	let dbStatus: DBStudioStatus | null = null;
	let httpRoutes: ProjectHTTPRoute[] = [];
	let supportingSummaryLoaded = false;
	let overviewInFlight = false;
	let environmentDialogOpen = false;
	let error = '';

	$: base = `/projects/${$page.params.id}`;
	$: latestDeployment = deployments[0] ?? null;
	$: operationalState = project
		? deriveProjectOperationalState({
			project,
			latestDeployment,
			runtimeEvidence: project.deployMode === 'static' ? 'not_applicable' : undefined
		})
		: null;
	$: attentionActionHref = operationalState?.primaryAction === 'view_logs'
		? `${base}/logs`
		: '';
	$: attentionHeadline = operationalState?.headline === 'Deploying'
		? 'Deployment in progress'
		: (operationalState?.headline ?? '');
	$: runtimeLabel = project
		? project.deployMode === 'compose'
			? 'Docker Compose'
			: project.deployMode === 'dockerfile'
				? 'Dockerfile'
				: project.deployMode === 'static'
					? 'Static site'
					: 'Container image'
		: '-';
	$: databaseLabel = dbStatus?.configured
		? `${dbStatus.connection?.driver?.toUpperCase() ?? 'Database'} · ${dbStatus.connected ? 'Connected' : 'Unavailable'}`
		: supportingSummaryLoaded
			? 'Not configured'
			: 'Checking…';
	$: additionalEndpoints = project
		? httpRoutes.map((route) => ({
			...route,
			url: projectRouteURL(project!.subdomain || project!.name, route.name, $page.url.protocol, $page.url.hostname)
		}))
		: [];
	$: primaryMetric = project ? selectPrimaryProjectMetric($projectStreamMetrics, project.mainService) : null;
	$: usageLabel = primaryMetric
		? `${formatMetricValue(primaryMetric.memoryMb)} MB · ${primaryMetric.cpu.toFixed(1)}% CPU`
		: 'Waiting for telemetry';
	$: usageDetail = primaryMetric?.uptime ? `Up ${primaryMetric.uptime}` : 'Live runtime usage';

	onMount(() => {
		void loadOverview();
		void loadSupportingSummary();

		const overviewInterval = setInterval(() => void loadOverview(true), 5000);
		return () => {
			clearInterval(overviewInterval);
		};
	});

	async function loadOverview(background = false) {
		if (overviewInFlight) return;
		overviewInFlight = true;
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
			overviewInFlight = false;
		}
	}

	async function loadSupportingSummary() {
		const projectId = $page.params.id ?? '';
		const [envResult, databaseResult, routeResult] = await Promise.allSettled([
			api.env.list(projectId),
			api.dbStudio.status(projectId),
			api.projects.routes(projectId)
		]);

		envCount = envResult.status === 'fulfilled' ? envResult.value.length : null;
		dbStatus = databaseResult.status === 'fulfilled' ? databaseResult.value : null;
		httpRoutes = routeResult.status === 'fulfilled' ? routeResult.value : [];
		supportingSummaryLoaded = true;
	}

	function formatMetricValue(value: number) {
		if (!Number.isFinite(value)) return '0';
		return value >= 100 ? value.toFixed(0) : value.toFixed(1);
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
</script>

<svelte:head>
	<title>{project?.name ?? 'Project'} · MyPaaS</title>
</svelte:head>

{#if error && !project}
	<div class="workspace-section">
		<ErrorState title="Could not load project dashboard" message={error} on:retry={() => void loadOverview()} />
	</div>
{:else if project}
	<div class="project-overview-workspace">
		{#if error}
			<div class="border-b border-amber-200 bg-amber-50 px-5 py-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-200" role="alert">
				<p class="font-medium">Dashboard refresh failed</p>
				<p class="mt-1">{error}</p>
			</div>
		{/if}

		{#if operationalState && operationalState.attention !== 'none'}
			<section class="workspace-section border-b border-gray-100/70 dark:border-neutral-900">
				<div class="flex flex-col gap-3 px-5 py-3 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0">
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">{attentionHeadline}</h2>
						<p class="mt-1 text-[13px] text-gray-600 dark:text-gray-300">{operationalState.detail}</p>
					</div>
					{#if attentionActionHref}
						<ActionLink href={attentionActionHref} variant="secondary" size="xs">
							<ArrowRight slot="icon" class="h-3.5 w-3.5" />
							{operationalState.primaryActionLabel}
						</ActionLink>
					{/if}
				</div>
			</section>
		{/if}

		<section class="workspace-section border-b border-gray-100/70 bg-gray-100/70 dark:border-neutral-900 dark:bg-neutral-900">
			<div class={`grid gap-px sm:grid-cols-2 ${project.deployMode === 'static' ? 'xl:grid-cols-3' : 'xl:grid-cols-4'}`}>
				{#if project.deployMode !== 'static'}
					<a href={`${base}/settings`} class="group min-w-0 bg-white px-4 py-3 hover:bg-gray-50/80 dark:bg-neutral-950 dark:hover:bg-neutral-900">
						<div class="flex items-center justify-between gap-3">
							<div class="flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400">
								<Settings2 class="h-3.5 w-3.5" aria-hidden="true" />
								Runtime
							</div>
							<ArrowRight class="h-3.5 w-3.5 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
						</div>
						<p class="mt-1.5 text-sm font-semibold text-gray-950 dark:text-white">{runtimeLabel}</p>
						<p class="mt-0.5 font-mono text-xs text-gray-500 dark:text-gray-400">:{project.appPort}</p>
					</a>

					<div class="min-w-0 bg-white px-4 py-3 dark:bg-neutral-950">
						<div class="flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400">
							<Activity class="h-3.5 w-3.5" aria-hidden="true" />
							Usage <span class="font-normal text-gray-400 dark:text-gray-500">Live</span>
						</div>
						<p class="metric-value mt-1.5 text-sm font-semibold text-gray-950 dark:text-white">{usageLabel}</p>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{usageDetail}</p>
					</div>
				{/if}

				<button type="button" class="group min-w-0 bg-white px-4 py-3 text-left hover:bg-gray-50/80 dark:bg-neutral-950 dark:hover:bg-neutral-900" on:click={() => (environmentDialogOpen = true)}>
					<div class="flex items-center justify-between gap-3">
						<div class="flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400">
							<KeyRound class="h-3.5 w-3.5" aria-hidden="true" />
							Environment
						</div>
						<ArrowRight class="h-3.5 w-3.5 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
					</div>
					<p class="mt-1.5 truncate text-sm font-semibold text-gray-950 dark:text-white">{envCount === null ? (supportingSummaryLoaded ? 'Unavailable' : 'Checking…') : `${envCount} variable${envCount === 1 ? '' : 's'}`}</p>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">configured</p>
				</button>

				<a href={`${base}/database`} class="group min-w-0 bg-white px-4 py-3 hover:bg-gray-50/80 dark:bg-neutral-950 dark:hover:bg-neutral-900">
					<div class="flex items-center justify-between gap-3">
						<div class="flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400">
							<Database class="h-3.5 w-3.5" aria-hidden="true" />
							Database
						</div>
						<ArrowRight class="h-3.5 w-3.5 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
					</div>
					<p class="mt-1.5 truncate text-sm font-semibold text-gray-950 dark:text-white">{databaseLabel}</p>
				</a>

				{#if project.deployMode === 'static'}
					<a href={`${base}/settings`} class="group min-w-0 bg-white px-4 py-3 hover:bg-gray-50/80 dark:bg-neutral-950 dark:hover:bg-neutral-900">
						<div class="flex items-center justify-between gap-3">
							<div class="flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400">
								<Settings2 class="h-3.5 w-3.5" aria-hidden="true" />
								Deployment limits
							</div>
							<ArrowRight class="h-3.5 w-3.5 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
						</div>
						<p class="metric-value mt-1.5 text-sm font-semibold text-gray-950 dark:text-white">{project.memoryLimitMb} MB · {project.cpuLimit} CPU</p>
					</a>
				{/if}
			</div>
		</section>

		{#if additionalEndpoints.length > 0}
			<section class="workspace-section border-b border-gray-100/70 dark:border-neutral-900">
				<div class="flex items-center justify-between gap-3 px-4 py-3">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Additional routes</h2>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Extra Compose HTTP endpoints for this project.</p>
					</div>
					<span class="text-xs text-gray-400 dark:text-gray-500">{additionalEndpoints.length} route{additionalEndpoints.length === 1 ? '' : 's'}</span>
				</div>
				<div class="grid border-t border-gray-100/70 bg-gray-100/70 dark:border-neutral-900 dark:bg-neutral-900 md:grid-cols-2">
					{#each additionalEndpoints as endpoint}
						<a href={endpoint.url} target="_blank" rel="noreferrer" class="group min-w-0 bg-white px-4 py-3 hover:bg-gray-50/80 dark:bg-neutral-950 dark:hover:bg-neutral-900">
							<div class="flex items-center justify-between gap-3">
								<div class="flex min-w-0 items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400">
									<ExternalLink class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
									<span class="truncate">{endpoint.name}</span>
								</div>
								<ArrowRight class="h-3.5 w-3.5 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
							</div>
							<p class="mt-1.5 truncate font-mono text-xs font-medium text-gray-950 dark:text-white">{endpoint.url}</p>
							<p class="mt-0.5 font-mono text-[11px] text-gray-400 dark:text-gray-500">{endpoint.service}:{endpoint.containerPort}</p>
						</a>
					{/each}
				</div>
			</section>
		{/if}

		<ProjectObservability {project} />

		<section class="workspace-section bg-white dark:bg-neutral-950">
			<div class="flex items-center justify-between gap-3 border-b border-gray-100/70 px-4 py-3 dark:border-neutral-900">
				<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Latest deployment</h2>
				<ActionLink href={`${base}/deployments`} variant="ghost" size="xs">
					<History slot="icon" class="h-3.5 w-3.5" />
					View all
				</ActionLink>
			</div>

			{#if latestDeployment}
				<div class="px-4 py-4">
					<div class="flex flex-wrap items-center justify-between gap-3">
						<div class="flex min-w-0 items-center gap-3">
							<StatusBadge
								status={latestDeployment.status}
								label={deploymentHistoryLabel(latestDeployment.status, latestDeployment.id, project.activeDeploymentId, project.status, project.deployMode)}
							/>
							<p class="truncate font-mono text-sm font-medium text-gray-950 dark:text-white">
								{project.sourceType === 'registry' ? (latestDeployment.imageTag ?? project.imageRef ?? '-') : (latestDeployment.commitSha?.slice(0, 8) ?? '-')}
							</p>
						</div>
						<p class="metric-value text-xs text-gray-500 dark:text-gray-400">{formatDuration(latestDeployment.startedAt, latestDeployment.finishedAt)}</p>
					</div>
					<p class="mt-3 text-sm text-gray-800 dark:text-gray-200">{latestDeployment.commitMessage || 'No deployment message'}</p>
					<div class="mt-3 flex flex-wrap gap-x-5 gap-y-1.5 text-xs text-gray-500 dark:text-gray-400">
						<span>Triggered by <span class="capitalize text-gray-700 dark:text-gray-300">{latestDeployment.triggeredBy}</span></span>
						<span>{formatDate(latestDeployment.startedAt)}</span>
					</div>
				</div>
			{:else}
				<div class="px-4 py-6">
					<EmptyState title="No deployment yet." description="Use Deploy above when you are ready to publish the first version." compact />
				</div>
			{/if}
		</section>
	</div>

	{#if environmentDialogOpen}
		<EnvironmentVariablesDialog
			projectId={project.id}
			fullPageHref={`${base}/env`}
			on:close={() => (environmentDialogOpen = false)}
			on:changed={(event) => (envCount = event.detail)}
		/>
	{/if}
{/if}
