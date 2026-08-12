<script lang="ts">
	import { ExternalLink, Play, Plus, RefreshCw, Search, Square, TriangleAlert, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import ActionLink from '$components/ActionLink.svelte';
	import CapacityMetricChart from '$components/CapacityMetricChart.svelte';
	import StatusBadge from '$components/StatusBadge.svelte';
	import FleetStatusChart from '$components/FleetStatusChart.svelte';
	import IconButton from '$components/IconButton.svelte';
	import PageHeader from '$components/PageHeader.svelte';
	import Pagination from '$components/Pagination.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import StatTile from '$components/StatTile.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { api, type HostStats } from '$api';
	import { toast } from '$stores/toast';
	import { selectPrimaryProjectMetric } from '$lib/utils/project-dashboard';
	import { projectURL } from '$lib/utils/urls';
	import type { Project, QuotaUsage } from '$types';

	type FleetSegment = {
		label: string;
		value: number;
		tone: 'success' | 'info' | 'warning' | 'danger' | 'neutral';
	};

	type DeployModeSegment = {
		label: string;
		value: number;
		barClass: string;
		textClass: string;
	};

	const pageSize = 20;
	let projects: Project[] = [];
	let quota: QuotaUsage | null = null;
	let hostStats: HostStats | null = null;
	let loading = true;
	let error = '';
	let projectActionId = '';
	let projectActionType: 'start' | 'stop' | 'deploy' | '' = '';
	let currentPage = 0;
	let searchQuery = '';
	let projectUptimes: Record<string, string> = {};
	let projectCpu: Record<string, number> = {};
	let projectMemory: Record<string, number> = {};
	let uptimeLoadingIds = new Set<string>();
	let uptimeRefreshToken = 0;
	let lastRefreshedAt: Date | null = null;
	let projectsInFlight = false;

	$: normalizedSearch = searchQuery.trim().toLowerCase();
	$: filteredProjects = normalizedSearch
		? projects.filter((project) =>
				[project.name, project.subdomain, project.repoUrl, project.imageRef ?? '', project.sourceType, project.branch, project.deployMode, project.mainService ?? '', project.status].join(' ').toLowerCase().includes(normalizedSearch)
			)
		: projects;
	$: hostRamMb = hostStats ? Math.round(hostStats.host_ram_bytes / (1024 * 1024)) : 0;
	$: hostRamWarning = hostStats && hostRamMb > 0 && hostStats.allocated_ram_mb > (hostRamMb * 0.85);
	$: memoryConfiguredPercent = quota && quota.memoryLimitMb > 0 ? Math.min(100, (quota.memoryUsedMb / quota.memoryLimitMb) * 100) : 0;
	$: memoryRuntimePercent = quota && quota.memoryUsedMb > 0 ? Math.min(100, (quota.memoryRuntimeMb / quota.memoryUsedMb) * 100) : 0;
	$: cpuConfiguredPercent = quota && quota.cpuLimit > 0 ? Math.min(100, (quota.cpuUsed / quota.cpuLimit) * 100) : 0;
	$: projectPercent = quota && quota.projectLimit > 0 ? Math.min(100, (quota.projectCount / quota.projectLimit) * 100) : 0;
	$: getDerivedStatus = (project: Project) => {
		if (project.status === 'running' && projectUptimes[project.id] === '-') return 'crashed';
		return project.status;
	};
	$: runningCount = projects.filter((project) => getDerivedStatus(project) === 'running').length;
	$: buildingCount = projects.filter((project) => getDerivedStatus(project) === 'building').length;
	$: issueCount = projects.filter((project) => getDerivedStatus(project) === 'crashed').length;
	$: stoppedCount = projects.filter((project) => getDerivedStatus(project) === 'stopped').length;
	$: pendingCount = projects.filter((project) => getDerivedStatus(project) === 'pending').length;
	$: dockerfileCount = projects.filter((project) => project.deployMode === 'dockerfile').length;
	$: composeCount = projects.filter((project) => project.deployMode === 'compose').length;
	$: staticCount = projects.filter((project) => project.deployMode === 'static').length;
	$: imageCount = projects.filter((project) => project.deployMode === 'image').length;
	$: latestProject = [...projects].sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())[0];
	$: healthyCopy = issueCount > 0 ? `${issueCount} project${issueCount !== 1 ? 's' : ''} need attention` : `${runningCount} running, no crashed projects`;
	$: syncLabel = error ? 'Sync needs attention' : loading ? 'Syncing workspace' : 'Workspace synced';
	$: syncDetail = lastRefreshedAt ? `Updated ${formatRefreshTime(lastRefreshedAt)}` : 'Waiting for first refresh';
	$: syncDotClass = error ? 'bg-amber-500' : loading ? 'bg-sky-500 animate-pulse' : 'bg-brand-500';
	$: healthSegments = [
		{ label: 'Running', value: runningCount, tone: 'success' },
		{ label: 'Building', value: buildingCount, tone: 'warning' },
		{ label: 'Stopped', value: stoppedCount, tone: 'neutral' },
		{ label: 'Pending', value: pendingCount, tone: 'info' },
		{ label: 'Crashed', value: issueCount, tone: 'danger' }
	] satisfies FleetSegment[];
	$: deployModeSegments = [
		{ label: 'Dockerfile', value: dockerfileCount, barClass: 'bg-sky-500', textClass: 'text-sky-700 dark:text-sky-300' },
		{ label: 'Compose', value: composeCount, barClass: 'bg-brand-500', textClass: 'text-brand-700 dark:text-brand-100' },
		{ label: 'Static', value: staticCount, barClass: 'bg-gray-400 dark:bg-gray-500', textClass: 'text-gray-600 dark:text-gray-300' },
		{ label: 'Image', value: imageCount, barClass: 'bg-amber-500', textClass: 'text-amber-700 dark:text-amber-300' }
	] satisfies DeployModeSegment[];
	$: deployModeTotal = deployModeSegments.reduce((sum, segment) => sum + segment.value, 0);
	$: dominantDeployMode = deployModeSegments.reduce((top, segment) => (segment.value > top.value ? segment : top), deployModeSegments[0]);
	$: maxPage = Math.max(0, Math.ceil(filteredProjects.length / pageSize) - 1);
	$: if (currentPage > maxPage) {
		currentPage = maxPage;
	}
	$: pageStart = currentPage * pageSize;
	$: visibleProjects = filteredProjects.slice(pageStart, pageStart + pageSize);
	$: hasNext = pageStart + pageSize < filteredProjects.length;
	$: if (!loading && visibleProjects.length > 0) {
		void loadUptimesFor(visibleProjects);
	}

	onMount(() => {
		void refreshDashboardData();
		const dashboardRefresh = setInterval(() => void loadProjects(true), 5000);
		return () => clearInterval(dashboardRefresh);
	});

	async function refreshDashboardData(background = false) {
		if (projectsInFlight) return;
		uptimeRefreshToken += 1;
		projectUptimes = {};
		projectCpu = {};
		projectMemory = {};
		uptimeLoadingIds = new Set();
		await loadProjects(background);
	}

	async function loadProjects(background = false) {
		if (projectsInFlight) return;
		projectsInFlight = true;
		if (!background) {
			loading = true;
		}
		error = '';
		try {
			[projects, quota, hostStats] = await Promise.all([api.projects.list(), api.me.quota(), api.admin.getHostStats()]);
			lastRefreshedAt = new Date();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load projects';
		} finally {
			if (!background) {
				loading = false;
			}
			projectsInFlight = false;
		}
	}

	function projectPrimaryAction(project: Project): 'start' | 'stop' | 'deploy' | 'busy' {
		const status = getDerivedStatus(project);
		if (status === 'building') return 'busy';
		if (status === 'running') return 'stop';
		if (status === 'stopped' || status === 'crashed') return 'start';
		return 'deploy';
	}

	function projectPrimaryLabel(project: Project) {
		const action = projectPrimaryAction(project);
		if (projectActionId === project.id) {
			if (projectActionType === 'stop') return 'Stopping project';
			if (projectActionType === 'start') return 'Starting project';
			return 'Deployment in progress';
		}
		if (action === 'busy') return 'Deployment in progress';
		if (action === 'stop') return 'Stop project';
		if (action === 'start') return 'Start project';
		return 'Deploy project';
	}

	function projectPrimaryVariant(project: Project): 'secondary' | 'primary' | 'ghostDanger' | 'ghost' {
		const action = projectPrimaryAction(project);
		if (action === 'stop') return 'ghostDanger';
		if (action === 'busy') return 'ghost';
		return 'primary';
	}

	async function handlePrimaryProjectAction(project: Project) {
		if (projectActionId) return;
		const action = projectPrimaryAction(project);
		if (action === 'busy') return;

		projectActionId = project.id;
		projectActionType = action;
		try {
			if (action === 'stop') {
				await api.projects.stop(project.id);
				toast.success(`${project.name} stopped`);
			} else if (action === 'start') {
				await api.projects.start(project.id);
				toast.success(`${project.name} started`);
			} else {
				await api.projects.deploy(project.id);
				toast.success(`Deployment queued for ${project.name}`);
			}
			await refreshDashboardData(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : `Failed to ${action} project`);
		} finally {
			projectActionId = '';
			projectActionType = '';
		}
	}

	function formatDate(value: string) {
		return new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
	}

	function formatDateTime(value: string) {
		return new Date(value).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
	}

	function formatRefreshTime(value: Date) {
		return value.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
	}

	function appUrl(project: Project) {
		return projectURL(project.subdomain, $page.url.protocol, $page.url.hostname);
	}

	function handleSearch(value: string) {
		searchQuery = value;
		currentPage = 0;
	}

	async function loadUptimesFor(rows: Project[]) {
		const pending = rows.filter((project) => !(project.id in projectUptimes) && !uptimeLoadingIds.has(project.id));
		if (pending.length === 0) return;

		const refreshToken = uptimeRefreshToken;
		uptimeLoadingIds = new Set([...uptimeLoadingIds, ...pending.map((project) => project.id)]);
		await Promise.all(
			pending.map(async (project) => {
				try {
					const snapshot = await api.metrics.snapshot(project.id);
					if (refreshToken !== uptimeRefreshToken) return;
					const metrics = selectPrimaryProjectMetric(snapshot, project.mainService);
					projectUptimes = { ...projectUptimes, [project.id]: metrics?.uptime ?? '-' };
					if (metrics) {
						projectCpu = { ...projectCpu, [project.id]: metrics.cpu };
						projectMemory = { ...projectMemory, [project.id]: metrics.memoryMb };
					}
				} catch {
					if (refreshToken !== uptimeRefreshToken) return;
					projectUptimes = { ...projectUptimes, [project.id]: '-' };
				} finally {
					if (refreshToken !== uptimeRefreshToken) return;
					const next = new Set(uptimeLoadingIds);
					next.delete(project.id);
					uptimeLoadingIds = next;
				}
			})
		);
	}
</script>

<svelte:head>
	<title>Projects · MyPaas</title>
</svelte:head>

<div class="page-shell py-6">
	<div class="mb-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
		<PageHeader
			title="Deployment control plane"
			description={`${projects.length} project${projects.length !== 1 ? 's' : ''} connected. Watch health, capacity, and deploy actions from one operational surface.`}
			className="!mb-0"
		/>

		<div class="flex w-full flex-col items-stretch gap-2 sm:w-auto sm:items-end">
			<div class="surface-muted flex min-h-10 w-full items-center gap-2 px-3 py-2 text-left sm:min-w-[15rem]">
				<span class={`h-2 w-2 shrink-0 rounded-full ${syncDotClass}`}></span>
				<div class="min-w-0">
					<p class="truncate text-xs font-medium text-gray-900 dark:text-white">{syncLabel}</p>
					<p class="truncate text-[11px] text-gray-500 dark:text-gray-400">{syncDetail}</p>
				</div>
			</div>
			<div class="flex justify-end gap-2">
				<IconButton label="Refresh dashboard data" variant="secondary" {loading} on:click={() => refreshDashboardData()}>
					<RefreshCw class="h-4 w-4" aria-hidden="true" />
				</IconButton>
				<ActionLink href="/projects/new" variant="primary">
					<Plus slot="icon" class="h-4 w-4" />
					New project
				</ActionLink>
			</div>
		</div>
	</div>

	{#if hostRamWarning}
		<div class="mb-6 rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200" role="alert">
			<div class="flex gap-3">
				<TriangleAlert class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
				<div>
					<p class="font-semibold">VM capacity reached</p>
					<p class="mt-1">
						Your VM ({hostRamMb}MB RAM) is almost fully allocated to projects ({hostStats?.allocated_ram_mb}MB used). Deploying more projects may cause Out-Of-Memory (OOM) crashes. Consider upgrading your VM or reducing project limits in Settings.
					</p>
				</div>
			</div>
		</div>
	{/if}

	<div class="mb-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
		<StatTile label="Fleet health" value={issueCount > 0 ? 'Attention' : 'Healthy'} detail={healthyCopy} tone={issueCount > 0 ? 'danger' : 'success'} />
		<StatTile label="Running now" value={String(runningCount)} detail={`${buildingCount} building, ${pendingCount} pending`} tone={buildingCount > 0 ? 'warning' : 'success'} />
		<StatTile
			label="Latest activity"
			value={latestProject?.name ?? 'No activity'}
			detail={latestProject ? formatDateTime(latestProject.updatedAt) : 'Create a project to start deploying'}
			tone="neutral"
		>
			{#if latestProject}
				<a href="/projects/{latestProject.id}" class="text-xs font-medium text-brand-700 hover:underline dark:text-brand-100"> Open project </a>
			{/if}
		</StatTile>
		<StatTile
			label="Project slots"
			value={quota ? `${quota.projectCount}/${quota.projectLimit}` : `${projects.length}`}
			detail={quota ? `${projectPercent.toFixed(0)}% of whitelist quota` : 'Waiting for quota data'}
			tone={projectPercent >= 80 ? 'warning' : 'info'}
		/>
	</div>

	<div class="mb-5 grid gap-4 xl:grid-cols-[minmax(0,1fr)_24rem]">
		<SectionPanel title="Capacity and deploy modes" description="Configured quota, live resource shape, and runtime composition across connected projects." contentClass="p-0">
			{#if quota}
				<div class="grid gap-px bg-gray-100 dark:bg-gray-800 sm:grid-cols-2 xl:grid-cols-5">
					{#if hostStats}
						<CapacityMetricChart
							label="Host RAM"
							value={`${hostStats.allocated_ram_mb}/${hostRamMb} MB`}
							detail={`${hostRamMb > 0 ? ((hostStats.allocated_ram_mb / hostRamMb) * 100).toFixed(0) : 0}% allocated`}
							percent={hostRamMb > 0 ? Math.min(100, (hostStats.allocated_ram_mb / hostRamMb) * 100) : 0}
							tone={hostRamWarning ? 'danger' : 'neutral'}
							className="bg-white dark:bg-gray-900"
						/>
					{/if}
					<CapacityMetricChart
						label="Memory"
						value={`${quota.memoryUsedMb}/${quota.memoryLimitMb} MB`}
						detail={`${(quota.memoryLimitMb > 0 ? (quota.memoryUsedMb / quota.memoryLimitMb) * 100 : 0).toFixed(0)}% allocated`}
						percent={memoryConfiguredPercent}
						tone={memoryConfiguredPercent >= 80 ? 'warning' : 'neutral'}
						className="bg-white dark:bg-gray-900"
					/>
					<CapacityMetricChart
						label="CPU"
						value={`${quota.cpuUsed.toFixed(2)}/${quota.cpuLimit.toFixed(2)} cores`}
						detail={`${(quota.cpuLimit > 0 ? (quota.cpuUsed / quota.cpuLimit) * 100 : 0).toFixed(0)}% allocated`}
						percent={cpuConfiguredPercent}
						tone={cpuConfiguredPercent >= 80 ? 'warning' : 'info'}
						className="bg-white dark:bg-gray-900"
					/>
					<CapacityMetricChart
						label="Project slots"
						value={`${quota.projectCount}/${quota.projectLimit}`}
						detail="whitelist quota"
						percent={projectPercent}
						tone={projectPercent >= 80 ? 'warning' : 'success'}
						className="bg-white dark:bg-gray-900"
					/>
					<article class="min-w-0 bg-white p-4 dark:bg-gray-900" aria-label="Deploy mode composition">
						<div class="flex items-start justify-between gap-3">
							<div class="min-w-0">
								<p class="metric-label">Deploy modes</p>
								<p class="mt-1 truncate text-lg font-semibold tracking-tight text-gray-950 dark:text-white">
									{deployModeTotal > 0 ? dominantDeployMode.label : 'No modes'}
								</p>
							</div>
							<p class="shrink-0 font-mono text-xs font-semibold text-gray-600 dark:text-gray-300">{deployModeTotal} total</p>
						</div>

						<div class="mt-3 h-20 overflow-hidden rounded-md border border-gray-200 bg-white px-2 pb-2 pt-3 dark:border-gray-800 dark:bg-gray-950">
							<div class="flex h-full items-end gap-1.5">
								{#each deployModeSegments as segment}
									<div class="flex h-full min-w-0 flex-1 flex-col justify-end">
										<div
											class={`mx-auto w-full max-w-10 rounded-sm ${segment.barClass}`}
											style={`height: ${deployModeTotal > 0 && segment.value > 0 ? Math.max(12, (segment.value / deployModeTotal) * 100) : 0}%`}
											title={`${segment.label}: ${segment.value}`}
										></div>
									</div>
								{/each}
							</div>
						</div>

						<div class="mt-2 grid grid-cols-4 gap-2 text-[11px]">
							{#each deployModeSegments as segment}
								<div class="min-w-0">
									<p class={`truncate font-mono font-semibold ${segment.textClass}`}>{segment.value}</p>
									<p class="truncate text-gray-500 dark:text-gray-400">{segment.label}</p>
								</div>
							{/each}
						</div>
					</article>
				</div>
			{:else}
				<div class="grid gap-0 sm:grid-cols-2 xl:grid-cols-5" aria-busy="true">
					<div class="h-36 animate-pulse border-b border-gray-100 bg-gray-100/70 dark:border-gray-800 dark:bg-gray-800/60 sm:border-r xl:border-b-0"></div>
					<div class="h-36 animate-pulse border-b border-gray-100 bg-gray-100/70 dark:border-gray-800 dark:bg-gray-800/60 sm:border-r xl:border-b-0"></div>
					<div class="h-36 animate-pulse border-b border-gray-100 bg-gray-100/70 dark:border-gray-800 dark:bg-gray-800/60 sm:border-r xl:border-b-0"></div>
					<div class="h-36 animate-pulse border-b border-gray-100 bg-gray-100/70 dark:border-gray-800 dark:bg-gray-800/60 sm:border-r xl:border-b-0"></div>
					<div class="h-36 animate-pulse bg-gray-100/70 dark:bg-gray-800/60"></div>
				</div>
			{/if}
		</SectionPanel>

		<FleetStatusChart segments={healthSegments} title="Fleet health" subtitle="Status composition across connected projects." />
	</div>

	<TableShell
		title="Project inventory"
		description="Runtime state, deployment mode, capacity, and quick actions."
		{loading}
		loadingRows={3}
		error={error && projects.length === 0 ? error : ''}
		empty={filteredProjects.length === 0}
		emptyTitle={normalizedSearch ? 'No projects match this search' : 'No projects yet'}
		emptyDescription={normalizedSearch ? 'Try a project name, subdomain, branch, deploy mode, or status.' : 'Connect a Git repository or public container image and MyPaas will prepare the runtime.'}
		contentClass="overflow-hidden"
		on:retry={() => refreshDashboardData()}
	>
		<svelte:fragment slot="actions">
			<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
				<label class="relative block w-full sm:w-72">
					<span class="sr-only">Search projects</span>
					<Search class="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					<input
						type="text"
						inputmode="search"
						value={searchQuery}
						on:input={(event) => handleSearch((event.currentTarget as HTMLInputElement).value)}
						placeholder="Search projects"
						class="field h-9 w-full !pl-10 !pr-11"
					/>
					{#if searchQuery}
						<button
							type="button"
							on:click={() => handleSearch('')}
							class="app-focus absolute right-1 top-1/2 inline-flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-gray-500 dark:hover:bg-gray-800 dark:hover:text-gray-200"
							aria-label="Clear search"
							title="Clear search"
						>
							<X class="h-4 w-4" aria-hidden="true" />
						</button>
					{/if}
				</label>
				<IconButton label="Refresh dashboard data" variant="ghost" {loading} on:click={() => refreshDashboardData()}>
					<RefreshCw class="h-4 w-4" aria-hidden="true" />
				</IconButton>
			</div>
		</svelte:fragment>

		<svelte:fragment slot="notice">
			{#if error}
				<div
					role="alert"
					class="flex flex-wrap items-center justify-between gap-3 border-b border-amber-200 bg-amber-50 px-5 py-2 text-xs text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-200"
				>
					<span class="min-w-0 flex-1">
						{error}
					</span>
					<ActionButton variant="ghost" size="xs" on:click={() => refreshDashboardData()} {loading} loadingLabel="Retrying...">Retry</ActionButton>
				</div>
			{/if}
		</svelte:fragment>

		<div
			class="hidden w-full grid-cols-[minmax(0,1.35fr)_minmax(0,0.8fr)_minmax(0,1.35fr)_minmax(0,1.05fr)_minmax(0,0.7fr)_minmax(0,0.55fr)_minmax(0,0.55fr)_4.75rem] items-center gap-x-4 border-b border-gray-100 bg-gray-50/70 px-4 py-2 text-xs font-medium text-gray-500 dark:border-gray-800 dark:bg-gray-900/70 dark:text-gray-400 lg:grid"
		>
			<span>Project</span>
			<span>Status</span>
			<span>App URL</span>
			<span>Runtime</span>
			<span>Live Usage</span>
			<span>Uptime</span>
			<span>Updated</span>
			<span class="text-right">Actions</span>
		</div>
		<div class="divide-y divide-gray-100 dark:divide-gray-800">
			{#each visibleProjects as project}
				<div
					class="grid gap-y-3 px-4 py-4 transition-colors hover:bg-gray-50/80 dark:hover:bg-gray-900/70 lg:w-full lg:grid-cols-[minmax(0,1.35fr)_minmax(0,0.8fr)_minmax(0,1.35fr)_minmax(0,1.05fr)_minmax(0,0.7fr)_minmax(0,0.55fr)_minmax(0,0.55fr)_4.75rem] lg:items-center lg:gap-x-4"
				>
					<div class="min-w-0">
						<a href="/projects/{project.id}" class="block truncate text-sm font-semibold text-gray-950 hover:underline dark:text-white">
							{project.name}
						</a>
						<p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">
							{project.subdomain}
						</p>
					</div>
					<div><StatusBadge status={getDerivedStatus(project)} pulse /></div>
					<a href={appUrl(project)} target="_blank" rel="noopener" class="truncate font-mono text-xs text-gray-600 hover:text-gray-950 hover:underline dark:text-gray-300 dark:hover:text-white">
						{appUrl(project).replace(/^https?:\/\//, '')}
					</a>
					<div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
						<span class="rounded border border-gray-200 px-1.5 py-0.5 font-mono dark:border-gray-800">{project.deployMode}</span>
						{#if project.mainService}
							<span class="truncate">{project.mainService}</span>
						{/if}
						<span>{project.memoryLimitMb}MB</span>
					</div>
					<div class="font-mono text-xs text-gray-500 dark:text-gray-400">
						{#if project.id in projectMemory}
							{projectMemory[project.id].toFixed(0)}MB / {projectCpu[project.id].toFixed(1)}%
						{:else if uptimeLoadingIds.has(project.id)}
							Loading
						{:else}
							-
						{/if}
					</div>
					<div class="font-mono text-xs text-gray-500 dark:text-gray-400">
						{projectUptimes[project.id] ?? (uptimeLoadingIds.has(project.id) ? 'Loading' : '-')}
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">
						{formatDate(project.updatedAt)}
					</div>
					<div class="flex items-center justify-start gap-1.5 lg:justify-end">
						<IconButton label="Open project" href="/projects/{project.id}" variant="secondary">
							<ExternalLink class="h-4 w-4" aria-hidden="true" />
						</IconButton>
						<IconButton
							label={projectPrimaryLabel(project)}
							variant={projectPrimaryVariant(project)}
							on:click={() => handlePrimaryProjectAction(project)}
							loading={projectActionId === project.id || projectPrimaryAction(project) === 'busy'}
							disabled={(projectActionId !== '' && projectActionId !== project.id) || projectPrimaryAction(project) === 'busy'}
						>
							{#if projectPrimaryAction(project) === 'stop'}
								<Square class="h-4 w-4" aria-hidden="true" />
							{:else}
								<Play class="h-4 w-4" aria-hidden="true" />
							{/if}
						</IconButton>
					</div>
				</div>
			{/each}
		</div>
		<svelte:fragment slot="footer">
			<Pagination bind:page={currentPage} {pageSize} totalShown={visibleProjects.length} {hasNext} {loading} label="Projects" />
		</svelte:fragment>
	</TableShell>
</div>