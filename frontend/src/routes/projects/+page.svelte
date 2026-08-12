<script lang="ts">
	import { ExternalLink, FolderGit2, GitBranch, Package, Play, Plus, RefreshCw, Rocket, Search, Square, TriangleAlert, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import ActionLink from '$components/ActionLink.svelte';
	import CapacityMetricChart from '$components/CapacityMetricChart.svelte';
	import GitHubMark from '$components/GitHubMark.svelte';
	import Pagination from '$components/Pagination.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { api, type HostStats } from '$api';
	import { toast } from '$stores/toast';
	import { appendRollingSample, boundedPercent, deriveNetworkRate, type NetworkCounterSample, type NetworkRate } from '$lib/utils/host-telemetry';
	import { selectPrimaryProjectMetric } from '$lib/utils/project-dashboard';
	import { describeProjectSource, type RepositoryHost } from '$lib/utils/repository';
	import { projectURL } from '$lib/utils/urls';
	import type { Project } from '$types';

	const pageSize = 20;
	const telemetrySamples = 24;

	let projects: Project[] = [];
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
	let ramSeries: number[] = [];
	let cpuSeries: number[] = [];
	let storageSeries: number[] = [];
	let networkSeries: number[] = [];
	let networkBaseline: NetworkCounterSample | null = null;
	let currentNetworkRate: NetworkRate | null = null;

	$: normalizedSearch = searchQuery.trim().toLowerCase();
	$: filteredProjects = normalizedSearch
		? projects.filter((project) =>
				[project.name, project.subdomain, project.repoUrl, project.imageRef ?? '', project.sourceType, project.branch, project.deployMode, project.mainService ?? '', project.status].join(' ').toLowerCase().includes(normalizedSearch)
			)
		: projects;
	$: hostRamMb = hostStats ? hostStats.host_ram_bytes / (1024 * 1024) : 0;
	$: ramAllocationPercent = hostStats ? boundedPercent(hostStats.allocated_ram_mb, hostRamMb) : 0;
	$: cpuAllocationPercent = hostStats ? boundedPercent(hostStats.allocated_cpu, hostStats.host_cpu_cores) : 0;
	$: storageUsedBytes = hostStats?.storage ? Math.max(0, hostStats.storage.total_bytes - hostStats.storage.available_bytes) : 0;
	$: storagePercent = hostStats?.storage ? boundedPercent(storageUsedBytes, hostStats.storage.total_bytes) : 0;
	$: hostRamWarning = ramAllocationPercent >= 85;
	$: storageWarning = Boolean(hostStats?.storage && storagePercent >= 85);
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
	$: syncLabel = error ? 'Refresh needs attention' : loading ? 'Refreshing' : 'Up to date';
	$: syncDotClass = error ? 'bg-amber-500' : loading ? 'bg-gray-400 animate-pulse' : 'bg-gray-500 dark:bg-gray-400';
	$: maxPage = Math.max(0, Math.ceil(filteredProjects.length / pageSize) - 1);
	$: if (currentPage > maxPage) currentPage = maxPage;
	$: pageStart = currentPage * pageSize;
	$: visibleProjects = filteredProjects.slice(pageStart, pageStart + pageSize);
	$: hasNext = pageStart + pageSize < filteredProjects.length;
	$: if (!loading && visibleProjects.length > 0) void loadUptimesFor(visibleProjects);

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
		if (!background) loading = true;
		error = '';
		try {
			const [projectRows, nextHostStats] = await Promise.all([api.projects.list(), api.admin.getHostStats()]);
			projects = projectRows;
			hostStats = nextHostStats;
			recordHostTelemetry(nextHostStats, Date.now());
			lastRefreshedAt = new Date();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load projects';
		} finally {
			if (!background) loading = false;
			projectsInFlight = false;
		}
	}

	function recordHostTelemetry(stats: HostStats, sampledAtMs: number) {
		const totalRamMb = stats.host_ram_bytes / (1024 * 1024);
		ramSeries = appendRollingSample(ramSeries, boundedPercent(stats.allocated_ram_mb, totalRamMb), telemetrySamples);
		cpuSeries = appendRollingSample(cpuSeries, boundedPercent(stats.allocated_cpu, stats.host_cpu_cores), telemetrySamples);

		if (stats.storage && stats.storage.total_bytes > 0) {
			const used = Math.max(0, stats.storage.total_bytes - stats.storage.available_bytes);
			storageSeries = appendRollingSample(storageSeries, boundedPercent(used, stats.storage.total_bytes), telemetrySamples);
		} else {
			storageSeries = [];
		}

		if (stats.network) {
			const current: NetworkCounterSample = {
				interface: stats.network.interface,
				rxBytes: stats.network.rx_bytes,
				txBytes: stats.network.tx_bytes,
				sampledAtMs
			};
			currentNetworkRate = deriveNetworkRate(networkBaseline, current);
			networkBaseline = current;
			if (currentNetworkRate) {
				networkSeries = appendRollingSample(networkSeries, currentNetworkRate.totalBytesPerSecond, telemetrySamples);
			}
		} else {
			networkBaseline = null;
			currentNetworkRate = null;
			networkSeries = [];
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
			if (projectActionType === 'stop') return 'Stopping';
			if (projectActionType === 'start') return 'Starting';
			return 'Deploying';
		}
		if (action === 'busy') return 'Deploying';
		if (action === 'stop') return 'Stop';
		if (action === 'start') return 'Start';
		return 'Deploy';
	}

	function projectPrimaryVariant(project: Project): 'secondary' | 'primary' | 'ghostDanger' | 'ghost' {
		const action = projectPrimaryAction(project);
		if (action === 'stop') return 'ghostDanger';
		if (action === 'busy') return 'ghost';
		return 'primary';
	}

	function projectPrimaryIcon(project: Project) {
		const action = projectPrimaryAction(project);
		if (action === 'stop') return Square;
		if (action === 'deploy' || action === 'busy') return Rocket;
		return Play;
	}

	function runtimeLabel(project: Project) {
		if (project.deployMode === 'compose') return 'Docker Compose';
		if (project.deployMode === 'dockerfile') return 'Dockerfile';
		if (project.deployMode === 'static') return 'Static';
		return 'Container Image';
	}

	function sourceIcon(host: RepositoryHost) {
		if (host === 'github') return GitHubMark;
		if (host === 'registry') return Package;
		return FolderGit2;
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

	function formatRefreshTime(value: Date) {
		return value.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
	}

	function formatBytes(value: number) {
		if (!Number.isFinite(value) || value < 0) return '-';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		let amount = value;
		let index = 0;
		while (amount >= 1024 && index < units.length - 1) {
			amount /= 1024;
			index += 1;
		}
		const digits = amount >= 100 || index === 0 ? 0 : amount >= 10 ? 1 : 2;
		return `${amount.toFixed(digits)} ${units[index]}`;
	}

	function formatRate(value: number | null | undefined) {
		return value === null || value === undefined ? '-' : `${formatBytes(value)}/s`;
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
	<div class="mb-5 flex flex-wrap items-center justify-between gap-3">
		<p class="inline-flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400" aria-live="polite">
			<span class={`status-dot ${syncDotClass}`}></span>
			{syncLabel}{lastRefreshedAt ? ` · ${formatRefreshTime(lastRefreshedAt)}` : ''}
			<span class="text-gray-400 dark:text-gray-500">· {projects.length} project{projects.length === 1 ? '' : 's'}</span>
		</p>
		<div class="flex items-center gap-2">
			<ActionButton variant="secondary" loading={loading} loadingLabel="Refreshing" on:click={() => refreshDashboardData()}>
				<RefreshCw slot="icon" class="h-4 w-4" />
				Refresh
			</ActionButton>
			<ActionLink href="/projects/new" variant="primary">
				<Plus slot="icon" class="h-4 w-4" />
				New project
			</ActionLink>
		</div>
	</div>

	{#if hostRamWarning || storageWarning}
		<div class="alert-warning mb-5" role="alert">
			<TriangleAlert class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
			<div>
				<p class="font-semibold">Host capacity needs attention</p>
				<p class="mt-1">
					{#if hostRamWarning}RAM allocation is {ramAllocationPercent.toFixed(0)}%.{/if}
					{#if storageWarning} Storage usage is {storagePercent.toFixed(0)}%.{/if}
					 Keep enough headroom for builds, runtime spikes, and platform services.
				</p>
			</div>
		</div>
	{/if}

	<SectionPanel title="Host resources" description="Real rolling samples from MyPaaS capacity data and host telemetry. RAM and CPU values represent allocation, not live host utilization." contentClass="p-0" className="mb-5">
		{#if hostStats}
			<div class="grid gap-px bg-gray-100 dark:bg-neutral-800 sm:grid-cols-2 xl:grid-cols-4">
				<CapacityMetricChart
					label="RAM allocation"
					value={`${hostStats.allocated_ram_mb.toFixed(0)} / ${hostRamMb.toFixed(0)} MB`}
					indicator={`${ramAllocationPercent.toFixed(0)}%`}
					detail={`${ramAllocationPercent.toFixed(0)}% allocated`}
					series={ramSeries}
					resource="memory"
					className="bg-white dark:bg-neutral-900"
				/>
				<CapacityMetricChart
					label="CPU allocation"
					value={`${hostStats.allocated_cpu.toFixed(2)} / ${hostStats.host_cpu_cores.toFixed(2)} cores`}
					indicator={`${cpuAllocationPercent.toFixed(0)}%`}
					detail={`${cpuAllocationPercent.toFixed(0)}% allocated`}
					series={cpuSeries}
					resource="cpu"
					className="bg-white dark:bg-neutral-900"
				/>
				<CapacityMetricChart
					label="Storage"
					value={hostStats.storage ? `${formatBytes(storageUsedBytes)} / ${formatBytes(hostStats.storage.total_bytes)}` : 'Unavailable'}
					indicator={hostStats.storage ? `${storagePercent.toFixed(0)}%` : ''}
					detail={hostStats.storage ? `${formatBytes(hostStats.storage.available_bytes)} available` : 'Requires Phase 6 host telemetry'}
					series={storageSeries}
					resource="storage"
					className="bg-white dark:bg-neutral-900"
				/>
				<CapacityMetricChart
					label="Network"
					value={currentNetworkRate ? formatRate(currentNetworkRate.totalBytesPerSecond) : hostStats.network ? 'Collecting…' : 'Unavailable'}
					indicator={hostStats.network?.interface ?? ''}
					detail={currentNetworkRate ? `↓ ${formatRate(currentNetworkRate.rxBytesPerSecond)} · ↑ ${formatRate(currentNetworkRate.txBytesPerSecond)}` : hostStats.network ? 'Waiting for the next counter sample' : 'Requires Phase 6 host telemetry'}
					series={networkSeries}
					resource="network"
					maxValue={null}
					rangeLabel="auto scale"
					className="bg-white dark:bg-neutral-900"
				/>
			</div>
		{:else}
			<div class="grid sm:grid-cols-2 xl:grid-cols-4" aria-busy="true">
				{#each Array(4) as _}
					<div class="h-40 animate-pulse border-b border-gray-100 bg-gray-100/70 last:border-b-0 dark:border-neutral-800 dark:bg-neutral-800/60 sm:border-r xl:border-b-0"></div>
				{/each}
			</div>
		{/if}
	</SectionPanel>

	<TableShell
		title="Inventory"
		description="Runtime shape, current service sample, and the next relevant action for each project."
		{loading}
		loadingRows={3}
		error={error && projects.length === 0 ? error : ''}
		empty={filteredProjects.length === 0}
		emptyTitle={normalizedSearch ? 'No projects match this search' : 'No projects yet'}
		emptyDescription={normalizedSearch ? 'Try a project name, subdomain, branch, deploy mode, or status.' : 'Connect a Git repository or public container image and MyPaaS will prepare the runtime.'}
		contentClass="overflow-hidden"
		on:retry={() => refreshDashboardData()}
	>
		<svelte:fragment slot="actions">
			<label class="relative block w-full sm:w-72">
				<span class="sr-only">Search projects</span>
				<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400 dark:text-gray-500" aria-hidden="true" />
				<input
					type="text"
					inputmode="search"
					value={searchQuery}
					on:input={(event) => handleSearch((event.currentTarget as HTMLInputElement).value)}
					placeholder="Search projects"
					class="field h-9 w-full !pl-9"
				/>
				{#if searchQuery}
					<button type="button" on:click={() => handleSearch('')} class="app-focus absolute right-1 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-gray-500 dark:hover:bg-neutral-800 dark:hover:text-gray-200" aria-label="Clear search" title="Clear search">
						<X class="h-4 w-4" aria-hidden="true" />
					</button>
				{/if}
			</label>
		</svelte:fragment>

		<svelte:fragment slot="notice">
			{#if error}
				<div role="alert" class="flex flex-wrap items-center justify-between gap-3 border-b border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-200">
					<span class="min-w-0 flex-1">{error}</span>
					<ActionButton variant="ghost" size="xs" on:click={() => refreshDashboardData()} {loading} loadingLabel="Retrying">
						<RefreshCw slot="icon" class="h-3.5 w-3.5" />
						Retry
					</ActionButton>
				</div>
			{/if}
		</svelte:fragment>

		<table class="data-table hidden table-fixed lg:table">
			<colgroup>
				<col class="w-[20%]" />
				<col class="w-[16%]" />
				<col class="w-[11%]" />
				<col class="w-[13%]" />
				<col class="w-[11%]" />
				<col class="w-[8%]" />
				<col class="w-[8%]" />
				<col class="w-[13%]" />
			</colgroup>
			<thead>
				<tr>
					<th>Project</th>
					<th>Repository</th>
					<th>Branch</th>
					<th>Runtime</th>
					<th>Usage</th>
					<th>Uptime</th>
					<th>Updated</th>
					<th class="text-right">Action</th>
				</tr>
			</thead>
			<tbody>
				{#each visibleProjects as project}
					{@const source = describeProjectSource(project)}
					{@const crashed = getDerivedStatus(project) === 'crashed'}
					<tr>
						<td>
							<div class="min-w-0">
								<a href="/projects/{project.id}" class="inline-flex max-w-full items-center gap-1.5 truncate text-sm font-medium text-gray-950 hover:underline dark:text-white">
									{#if crashed}<span class="status-dot bg-red-500" title="Crashed"></span>{/if}
									<span class="truncate">{project.name}</span>
								</a>
								<a href={appUrl(project)} target="_blank" rel="noopener" class="mt-0.5 inline-flex max-w-full items-center gap-1 truncate font-mono text-xs text-gray-500 hover:text-gray-950 hover:underline dark:text-gray-400 dark:hover:text-white">
									<span class="truncate">{appUrl(project).replace(/^https?:\/\//, '')}</span>
									<ExternalLink class="h-3 w-3 shrink-0" aria-hidden="true" />
								</a>
							</div>
						</td>
						<td>
							{#if source.href}
								<a href={source.href} target="_blank" rel="noopener" title={source.label} class="inline-flex max-w-full items-center gap-1.5 truncate text-sm text-gray-700 hover:text-gray-950 hover:underline dark:text-gray-300 dark:hover:text-white">
									<svelte:component this={sourceIcon(source.host)} class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
									<span class="truncate">{source.label}</span>
								</a>
							{:else}
								<span title={source.label} class="inline-flex max-w-full items-center gap-1.5 truncate text-sm text-gray-700 dark:text-gray-300">
									<svelte:component this={sourceIcon(source.host)} class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
									<span class="truncate">{source.label}</span>
								</span>
							{/if}
						</td>
						<td>
							{#if project.sourceType === 'git' && project.branch}
								<span class="inline-flex max-w-full items-center gap-1.5 truncate font-mono text-xs text-gray-600 dark:text-gray-300" title={project.branch}>
									<GitBranch class="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
									<span class="truncate">{project.branch}</span>
								</span>
							{:else}
								<span class="text-sm text-gray-400 dark:text-gray-500">—</span>
							{/if}
						</td>
						<td>
							<div class="min-w-0">
								<p class="truncate text-sm text-gray-800 dark:text-gray-200">{runtimeLabel(project)}</p>
								<p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">{project.mainService ? `${project.mainService} · ` : ''}{project.memoryLimitMb} MB</p>
							</div>
						</td>
						<td class="metric-value">
							{#if project.id in projectMemory}
								<span class="text-sm text-gray-800 dark:text-gray-200">{projectMemory[project.id].toFixed(0)} MB · {projectCpu[project.id].toFixed(1)}%</span>
							{:else if uptimeLoadingIds.has(project.id)}
								<span class="text-xs text-gray-400 dark:text-gray-500">Loading…</span>
							{:else}
								<span class="text-sm text-gray-400 dark:text-gray-500">-</span>
							{/if}
						</td>
						<td class="metric-value text-sm text-gray-800 dark:text-gray-200">{projectUptimes[project.id] ?? (uptimeLoadingIds.has(project.id) ? '…' : '-')}</td>
						<td class="text-xs text-gray-500 dark:text-gray-400">{formatDate(project.updatedAt)}</td>
						<td class="text-right">
							<ActionButton
								variant={projectPrimaryVariant(project)}
								size="xs"
								className="min-w-[6.25rem]"
								on:click={() => handlePrimaryProjectAction(project)}
								loading={projectActionId === project.id || projectPrimaryAction(project) === 'busy'}
								loadingLabel={projectPrimaryLabel(project)}
								disabled={(projectActionId !== '' && projectActionId !== project.id) || projectPrimaryAction(project) === 'busy'}
							>
								<svelte:component this={projectPrimaryIcon(project)} class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
								{projectPrimaryLabel(project)}
							</ActionButton>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>

		<div class="divide-y divide-gray-100 dark:divide-neutral-800 lg:hidden">
			{#each visibleProjects as project}
				{@const source = describeProjectSource(project)}
				{@const crashed = getDerivedStatus(project) === 'crashed'}
				<div class="px-4 py-3">
					<div class="flex items-center justify-between gap-3">
						<a href="/projects/{project.id}" class="inline-flex min-w-0 items-center gap-1.5 truncate text-sm font-medium text-gray-950 hover:underline dark:text-white">
							{#if crashed}<span class="status-dot bg-red-500" title="Crashed"></span>{/if}
							<span class="truncate">{project.name}</span>
						</a>
						<ActionButton
							variant={projectPrimaryVariant(project)}
							size="xs"
							on:click={() => handlePrimaryProjectAction(project)}
							loading={projectActionId === project.id || projectPrimaryAction(project) === 'busy'}
							loadingLabel={projectPrimaryLabel(project)}
							disabled={(projectActionId !== '' && projectActionId !== project.id) || projectPrimaryAction(project) === 'busy'}
						>
							<svelte:component this={projectPrimaryIcon(project)} class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
							{projectPrimaryLabel(project)}
						</ActionButton>
					</div>
					<a href={appUrl(project)} target="_blank" rel="noopener" class="mt-1 inline-flex max-w-full items-center gap-1 truncate font-mono text-xs text-gray-500 hover:text-gray-950 hover:underline dark:text-gray-400 dark:hover:text-white">
						<span class="truncate">{appUrl(project).replace(/^https?:\/\//, '')}</span>
						<ExternalLink class="h-3 w-3 shrink-0" aria-hidden="true" />
					</a>
					<div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
						{#if source.href}
							<a href={source.href} target="_blank" rel="noopener" class="inline-flex min-w-0 items-center gap-1 truncate hover:text-gray-950 hover:underline dark:hover:text-white">
								<svelte:component this={sourceIcon(source.host)} class="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
								<span class="truncate">{source.label}</span>
							</a>
						{:else}
							<span class="inline-flex min-w-0 items-center gap-1 truncate">
								<svelte:component this={sourceIcon(source.host)} class="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
								<span class="truncate">{source.label}</span>
							</span>
						{/if}
						{#if project.sourceType === 'git' && project.branch}
							<span class="inline-flex items-center gap-1 font-mono">
								<GitBranch class="h-3 w-3 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
								{project.branch}
							</span>
						{/if}
						<span>{runtimeLabel(project)}</span>
						<span class="metric-value">
							{#if project.id in projectMemory}
								{projectMemory[project.id].toFixed(0)} MB · {projectCpu[project.id].toFixed(1)}%
							{:else}
								—
							{/if}
						</span>
					</div>
				</div>
			{/each}
		</div>
		<svelte:fragment slot="footer">
			<Pagination bind:page={currentPage} {pageSize} totalShown={visibleProjects.length} {hasNext} {loading} label="Projects" />
		</svelte:fragment>
	</TableShell>

	<section class="surface mt-5 overflow-hidden">
		<div class="panel-header">
			<h2 class="panel-title">Fleet</h2>
			<p class="panel-description">Secondary fleet state and runtime mix across connected projects.</p>
		</div>
		<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 md:grid-cols-2 md:divide-x md:divide-y-0">
			<div class="p-4">
				<p class="metric-label">State</p>
				<div class="mt-2 flex flex-wrap gap-x-4 gap-y-2 text-sm text-gray-700 dark:text-gray-300">
					<span class="inline-flex items-center gap-2"><span class="status-dot bg-emerald-500"></span>{runningCount} running</span>
					<span class="inline-flex items-center gap-2"><span class="status-dot bg-amber-500"></span>{buildingCount} building</span>
					<span class="inline-flex items-center gap-2"><span class="status-dot bg-red-500"></span>{issueCount} crashed</span>
					<span class="inline-flex items-center gap-2"><span class="status-dot bg-gray-400 dark:bg-gray-500"></span>{stoppedCount} stopped</span>
					{#if pendingCount > 0}<span class="inline-flex items-center gap-2"><span class="status-dot bg-sky-500"></span>{pendingCount} pending</span>{/if}
				</div>
			</div>
			<div class="p-4">
				<p class="metric-label">Runtime mix</p>
				<div class="mt-2 flex flex-wrap gap-x-4 gap-y-2 text-sm text-gray-700 dark:text-gray-300">
					<span><strong class="metric-value font-semibold text-gray-950 dark:text-white">{composeCount}</strong> Compose</span>
					<span><strong class="metric-value font-semibold text-gray-950 dark:text-white">{dockerfileCount}</strong> Dockerfile</span>
					<span><strong class="metric-value font-semibold text-gray-950 dark:text-white">{staticCount}</strong> Static</span>
					<span><strong class="metric-value font-semibold text-gray-950 dark:text-white">{imageCount}</strong> Image</span>
				</div>
			</div>
		</div>
	</section>
</div>