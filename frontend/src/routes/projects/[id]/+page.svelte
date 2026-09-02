<script lang="ts">
	import { onMount } from 'svelte';
	import { ArrowRight, Database, ExternalLink, History, KeyRound, Settings2 } from '@lucide/svelte';
	import { page } from '$app/stores';
	import ActionLink from '$components/ActionLink.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ProjectObservability from '$components/ProjectObservability.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import ProjectStatus from '$components/ProjectStatus.svelte';
	import StatusBadge from '$components/StatusBadge.svelte';
	import { api, type ProjectHTTPRoute } from '$api';
	import { deploymentHistoryLabel } from '$lib/utils/deploymentHistory';
	import { deriveProjectOperationalState } from '$lib/utils/project-operational-state';
	import { projectRouteURL, projectURL } from '$lib/utils/urls';
	import type { DBStudioStatus, Deployment, Project } from '$types';

	let project: Project | null = null;
	let deployments: Deployment[] = [];
	let envCount: number | null = null;
	let dbStatus: DBStudioStatus | null = null;
	let httpRoutes: ProjectHTTPRoute[] = [];
	let supportingSummaryLoaded = false;
	let overviewInFlight = false;
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
		: operationalState?.primaryAction === 'view_deployment'
			? `${base}/deployments${latestDeployment ? `?focus=${encodeURIComponent(latestDeployment.id)}` : ''}`
			: '';
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
	$: primaryEndpoint = project
		? projectURL(project.subdomain || project.name, $page.url.protocol, $page.url.hostname)
		: '';
	$: additionalEndpoints = project
		? httpRoutes.map((route) => ({
			...route,
			url: projectRouteURL(project!.subdomain || project!.name, route.name, $page.url.protocol, $page.url.hostname)
		}))
		: [];

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
				<div class="flex flex-col gap-4 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-2">
							<ProjectStatus status={project.status} label={operationalState.statusLabel} tone={operationalState.statusTone} />
							<h2 class="text-base font-semibold text-gray-950 dark:text-white">{operationalState.headline}</h2>
						</div>
						<p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{operationalState.detail}</p>
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

		<ProjectObservability {project} />

		<div class="grid bg-gray-100/70 dark:bg-neutral-900 lg:grid-cols-[minmax(0,1.25fr)_minmax(20rem,0.75fr)]">
			<section class="min-w-0 bg-white dark:bg-neutral-950 lg:border-r lg:border-gray-100/70 lg:dark:border-neutral-900">
				<div class="panel-header flex items-start justify-between gap-3">
					<div>
						<h2 class="panel-title">Latest deployment</h2>
						<p class="panel-description">Latest deployment attempt.</p>
					</div>
					<ActionLink href={`${base}/deployments`} variant="ghost" size="xs">
						<History slot="icon" class="h-3.5 w-3.5" />
						View all
					</ActionLink>
				</div>

				{#if latestDeployment}
					<div class="p-5">
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
							<p class="metric-value text-[13px] text-gray-500 dark:text-gray-400">{formatDuration(latestDeployment.startedAt, latestDeployment.finishedAt)}</p>
						</div>
						<p class="mt-4 text-sm text-gray-800 dark:text-gray-200">{latestDeployment.commitMessage || 'No deployment message'}</p>
						<div class="mt-4 flex flex-wrap gap-x-6 gap-y-2 text-[13px] text-gray-500 dark:text-gray-400">
							<span>Triggered by <span class="capitalize text-gray-700 dark:text-gray-300">{latestDeployment.triggeredBy}</span></span>
							<span>{formatDate(latestDeployment.startedAt)}</span>
						</div>
					</div>
				{:else}
					<div class="px-5 py-8">
						<EmptyState title="No deployment yet." description="Use Deploy above when you are ready to publish the first version." compact />
					</div>
				{/if}
			</section>

			<section class="min-w-0 bg-white dark:bg-neutral-950">
				<div class="panel-header">
					<h2 class="panel-title">Project essentials</h2>
					<p class="panel-description">Endpoints and configuration.</p>
				</div>

				<div class="divide-y divide-gray-100/70 dark:divide-neutral-900">
					<div class="px-5 py-4">
						<div class="flex items-start gap-3">
							<ExternalLink class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0 flex-1">
								<p class="text-sm font-medium text-gray-950 dark:text-white">Public endpoints</p>
								<a href={primaryEndpoint} target="_blank" rel="noreferrer" class="mt-1 block truncate font-mono text-xs text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">{primaryEndpoint}</a>
								{#each additionalEndpoints as endpoint}
									<div class="mt-2">
										<a href={endpoint.url} target="_blank" rel="noreferrer" class="block truncate font-mono text-xs text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">{endpoint.url}</a>
										<p class="mt-0.5 font-mono text-xs text-gray-400 dark:text-gray-500">{endpoint.service}:{endpoint.containerPort} · HTTP route {endpoint.name}</p>
									</div>
								{/each}
							</div>
						</div>
					</div>

					<a href={`${base}/env`} class="group flex items-center justify-between gap-4 px-5 py-4 hover:bg-gray-50/70 dark:hover:bg-neutral-900/60">
						<div class="flex min-w-0 items-center gap-3">
							<KeyRound class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0">
								<p class="text-sm font-medium text-gray-950 dark:text-white">Environment</p>
								<p class="mt-1 text-[13px] text-gray-500 dark:text-gray-400">{envCount === null ? (supportingSummaryLoaded ? 'Unavailable' : 'Checking…') : `${envCount} variable${envCount === 1 ? '' : 's'} configured`}</p>
							</div>
						</div>
						<ArrowRight class="h-4 w-4 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
					</a>

					<a href={`${base}/database`} class="group flex items-center justify-between gap-4 px-5 py-4 hover:bg-gray-50/70 dark:hover:bg-neutral-900/60">
						<div class="flex min-w-0 items-center gap-3">
							<Database class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0">
								<p class="text-sm font-medium text-gray-950 dark:text-white">Database</p>
								<p class="mt-1 text-[13px] text-gray-500 dark:text-gray-400">{databaseLabel}</p>
							</div>
						</div>
						<ArrowRight class="h-4 w-4 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
					</a>

					<a href={`${base}/settings`} class="group flex items-center justify-between gap-4 px-5 py-4 hover:bg-gray-50/70 dark:hover:bg-neutral-900/60">
						<div class="flex min-w-0 items-center gap-3">
							<Settings2 class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0">
								<p class="text-sm font-medium text-gray-950 dark:text-white">Deployment setup</p>
								<p class="mt-1 truncate text-[13px] text-gray-500 dark:text-gray-400">{deploymentSummary}</p>
								<p class="metric-value mt-1 text-xs text-gray-400 dark:text-gray-500">{project.memoryLimitMb} MB · {project.cpuLimit} CPU</p>
							</div>
						</div>
						<ArrowRight class="h-4 w-4 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
					</a>
				</div>
			</section>
		</div>
	</div>
{/if}
