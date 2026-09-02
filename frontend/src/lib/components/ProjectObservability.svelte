<script lang="ts">
	import { onMount } from 'svelte';
	import { AlertTriangle, ChevronDown, LoaderCircle, Radio, SlidersHorizontal } from '@lucide/svelte';
	import { api } from '$api';
	import CloudflareChart from '$components/CloudflareChart.svelte';
	import CloudflareSetup from '$components/CloudflareSetup.svelte';
	import MultiServiceMetricChart from '$components/MultiServiceMetricChart.svelte';
	import { projectStreamConnection, projectStreamMetrics } from '$stores/project-stream';
	import { appendProjectMetricHistory, type ProjectMetricHistory } from '$lib/utils/project-metric-history';
	import type { CloudflareAnalytics, Project } from '$types';

	export let project: Project;

	let analytics: CloudflareAnalytics | null = null;
	let cloudflareConfigured: boolean | null = null;
	let analyticsLoading = true;
	let analyticsError = '';
	let metricHistory: ProjectMetricHistory = {};
	let lastHistorySample = '';
	let hiddenServices = new Set<string>();

	$: metricItems = $projectStreamMetrics?.items ?? [];
	$: services = Array.from(new Set(metricItems.map((item) => item.service).filter(Boolean)));
	$: visibleServices = services.filter((service) => !hiddenServices.has(service));
	$: visibleItems = metricItems.filter((item) => visibleServices.includes(item.service));
	$: currentSampleKey = $projectStreamMetrics?.collectedAt ?? '';
	$: if (currentSampleKey && currentSampleKey !== lastHistorySample) {
		metricHistory = appendProjectMetricHistory(metricHistory, metricItems);
		lastHistorySample = currentSampleKey;
	}
	$: cpuSeries = visibleItems.map((item) => ({
		service: item.service,
		values: metricHistory[item.service]?.cpu ?? [],
		value: `${item.cpu.toFixed(2)}%`
	}));
	$: memorySeries = visibleItems.map((item) => ({
		service: item.service,
		values: metricHistory[item.service]?.memoryMb ?? [],
		value: `${item.memoryMb.toFixed(1)} MB`
	}));
	$: sampleLabel = $projectStreamMetrics?.collectedAt ? new Date($projectStreamMetrics.collectedAt).toLocaleTimeString() : '';

	onMount(() => {
		void loadAnalytics();
		const analyticsInterval = setInterval(() => void loadAnalytics(true), 60_000);
		return () => clearInterval(analyticsInterval);
	});

	async function loadAnalytics(background = false) {
		if (!background && !analytics) analyticsLoading = true;
		if (cloudflareConfigured === null) {
			try {
				const settings = await api.admin.getSettings();
				cloudflareConfigured = !!settings.cloudflare_configured;
			} catch {
				cloudflareConfigured = null;
			}
		}
		if (cloudflareConfigured === false) {
			analyticsLoading = false;
			return;
		}
		try {
			analytics = await api.metrics.analytics(project.id);
			analyticsError = '';
			cloudflareConfigured = true;
		} catch (err) {
			analyticsError = err instanceof Error ? err.message : 'Edge analytics unavailable';
		} finally {
			analyticsLoading = false;
		}
	}

	function toggleService(service: string, visible: boolean) {
		const next = new Set(hiddenServices);
		if (visible) next.delete(service);
		else next.add(service);
		hiddenServices = next;
	}

	function showAllServices() {
		hiddenServices = new Set();
	}
</script>

<section aria-label="Runtime and edge observability">
	{#if analyticsLoading && !analytics}
		<div class="workspace-section flex items-center justify-between gap-4 border-b border-gray-100/70 px-4 py-3 dark:border-neutral-900">
			<div class="min-w-0">
				<p class="text-sm font-semibold text-gray-950 dark:text-white">Edge traffic</p>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Requests, bandwidth, and errors over the last 24 hours.</p>
			</div>
			<div class="flex shrink-0 items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
				<LoaderCircle class="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" aria-hidden="true" />
				Loading analytics…
			</div>
		</div>
	{:else if cloudflareConfigured === false}
		<details class="group workspace-section border-b border-gray-100/70 dark:border-neutral-900">
			<summary class="app-focus flex cursor-pointer list-none items-center justify-between gap-4 px-4 py-3 [&::-webkit-details-marker]:hidden">
				<div class="min-w-0">
					<div class="flex items-center gap-2">
						<p class="text-sm font-semibold text-gray-950 dark:text-white">Cloudflare analytics</p>
						<span class="rounded bg-gray-100 px-1.5 py-0.5 text-[11px] font-medium text-gray-500 dark:bg-neutral-900 dark:text-gray-400">Not configured</span>
					</div>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Optional edge traffic analytics. Runtime telemetry is unaffected.</p>
				</div>
				<div class="flex shrink-0 items-center gap-2 text-xs font-medium text-gray-700 dark:text-gray-300">
					Configure
					<ChevronDown class="h-3.5 w-3.5 transition-transform group-open:rotate-180" aria-hidden="true" />
				</div>
			</summary>
			<div class="border-t border-gray-100/70 bg-gray-50/40 p-4 dark:border-neutral-900 dark:bg-neutral-950/40">
				<CloudflareSetup on:success={() => { cloudflareConfigured = true; void loadAnalytics(); }} />
			</div>
		</details>
	{:else if analytics}
		<section class="workspace-section border-b border-gray-100/70 dark:border-neutral-900">
			<div class="flex items-center justify-between gap-3 px-4 py-3">
				<div>
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Edge traffic</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Last 24 hours</p>
				</div>
				<span class="text-xs text-gray-400 dark:text-gray-500">Cloudflare</span>
			</div>
			<div class="grid border-t border-gray-100/70 bg-gray-100/70 dark:border-neutral-900 dark:bg-neutral-900 sm:grid-cols-3">
				<div class="min-w-0 bg-white px-4 py-3 dark:bg-neutral-950 sm:border-r sm:border-gray-100/70 sm:dark:border-neutral-900">
					<p class="text-xs font-medium text-gray-500 dark:text-gray-400">Requests</p>
					<p class="metric-value mt-1 text-lg font-semibold text-gray-950 dark:text-white">{analytics.total_requests.toLocaleString()}</p>
					<div class="mt-1.5">
						{#if analytics.timeseries?.length > 0}
							<CloudflareChart data={analytics.timeseries} metric="requests" compact />
						{:else}
							<div class="flex h-14 items-center text-xs text-gray-400 dark:text-gray-500">Waiting for traffic samples.</div>
						{/if}
					</div>
				</div>

				<div class="min-w-0 bg-white px-4 py-3 dark:bg-neutral-950 sm:border-r sm:border-gray-100/70 sm:dark:border-neutral-900">
					<p class="text-xs font-medium text-gray-500 dark:text-gray-400">Bandwidth</p>
					<p class="metric-value mt-1 text-lg font-semibold text-gray-950 dark:text-white">{(analytics.bandwidth / (1024 * 1024)).toFixed(2)} <span class="text-xs font-medium text-gray-500">MB</span></p>
					<div class="mt-1.5">
						{#if analytics.timeseries?.length > 0}
							<CloudflareChart data={analytics.timeseries} metric="bandwidth" compact />
						{:else}
							<div class="flex h-14 items-center text-xs text-gray-400 dark:text-gray-500">Waiting for traffic samples.</div>
						{/if}
					</div>
				</div>

				<div class="min-w-0 bg-white px-4 py-3 dark:bg-neutral-950">
					<div class="flex items-center gap-2">
						<p class="text-xs font-medium text-gray-500 dark:text-gray-400">Edge errors</p>
						{#if analytics.errors > 0}<AlertTriangle class="h-3.5 w-3.5 text-amber-500" aria-hidden="true" />{/if}
					</div>
					<p class="metric-value mt-1 text-lg font-semibold text-gray-950 dark:text-white">{analytics.errors.toLocaleString()}</p>
					<div class="relative mt-1.5 flex h-14 items-center">
						{#if analytics.errors === 0}
							<div class="h-px w-full bg-gray-200 dark:bg-neutral-800"></div>
							<span class="absolute bottom-0 text-[11px] text-gray-400 dark:text-gray-500">No errors observed</span>
						{:else}
							<p class="text-xs text-gray-400 dark:text-gray-500">Aggregate only; interval error series is unavailable.</p>
						{/if}
					</div>
				</div>
			</div>
		</section>
	{:else if analyticsError}
		<div class="workspace-section flex items-center justify-between gap-4 border-b border-gray-100/70 px-4 py-3 dark:border-neutral-900">
			<div>
				<p class="text-sm font-semibold text-gray-950 dark:text-white">Edge traffic</p>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Edge analytics unavailable. {analyticsError}</p>
			</div>
		</div>
	{/if}

	{#if project.deployMode !== 'static'}
		<section class="workspace-section border-b border-gray-100/70 dark:border-neutral-900">
			<div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
				<div>
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Runtime</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">CPU and memory across running services.</p>
				</div>
				<div class="flex items-center gap-2">
					<div class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400" aria-live="polite">
						{#if $projectStreamConnection === 'connecting'}
							<LoaderCircle class="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" aria-hidden="true" /> Connecting…
						{:else if $projectStreamConnection === 'reconnecting'}
							<LoaderCircle class="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" aria-hidden="true" /> Reconnecting{sampleLabel ? ` · ${sampleLabel}` : '…'}
						{:else}
							<Radio class="h-3.5 w-3.5" aria-hidden="true" /> Live{sampleLabel ? ` · ${sampleLabel}` : ''}
						{/if}
					</div>
					{#if services.length > 1}
						<details class="group relative">
							<summary class="app-focus flex h-8 cursor-pointer list-none items-center gap-2 rounded-md border border-gray-200 bg-white px-3 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-neutral-800 dark:bg-neutral-950 dark:text-gray-300 dark:hover:bg-neutral-900 [&::-webkit-details-marker]:hidden">
								<SlidersHorizontal class="h-3.5 w-3.5" aria-hidden="true" />
								Services {visibleServices.length}/{services.length}
							</summary>
							<div class="overlay absolute right-0 z-30 mt-2 w-56 p-2">
								<button type="button" class="app-focus mb-1 flex w-full items-center rounded-md px-2.5 py-2 text-left text-[13px] font-medium text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-neutral-900" on:click={showAllServices}>
									All services
								</button>
								<div class="border-t border-gray-100 pt-1 dark:border-neutral-800">
									{#each services as service}
										<label class="flex cursor-pointer items-center gap-2 rounded-md px-2.5 py-2 text-[13px] text-gray-600 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-neutral-900">
											<input type="checkbox" checked={!hiddenServices.has(service)} on:change={(event) => toggleService(service, (event.currentTarget as HTMLInputElement).checked)} />
											<span class="truncate" title={service}>{service}</span>
										</label>
									{/each}
								</div>
							</div>
						</details>
					{/if}
				</div>
			</div>

			{#if metricItems.length > 0 && visibleItems.length > 0}
				<div class="grid gap-px border-t border-gray-100/70 bg-gray-100/70 dark:border-neutral-900 dark:bg-neutral-900 xl:grid-cols-2">
					<MultiServiceMetricChart label="CPU usage" series={cpuSeries} suffix="%" heightClass="h-14" compact />
					<MultiServiceMetricChart label="Memory usage" series={memorySeries} suffix=" MB" heightClass="h-14" compact />
				</div>
			{:else if metricItems.length > 0}
				<div class="border-t border-gray-100/70 px-4 py-4 text-xs text-gray-500 dark:border-neutral-900 dark:text-gray-400">
					All service lines are hidden. Use Services to show one or more.
				</div>
			{:else if $projectStreamConnection === 'connecting' || $projectStreamConnection === 'reconnecting'}
				<div class="flex items-center gap-2 border-t border-gray-100/70 px-4 py-4 text-xs text-gray-500 dark:border-neutral-900 dark:text-gray-400">
					<LoaderCircle class="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" aria-hidden="true" /> {$projectStreamConnection === 'reconnecting' ? 'Reconnecting…' : 'Connecting…'}
				</div>
			{:else}
				<div class="border-t border-gray-100/70 px-4 py-4 text-xs text-gray-500 dark:border-neutral-900 dark:text-gray-400">
					No active runtime metrics. CPU and memory appear while application services are running.
				</div>
			{/if}
		</section>
	{/if}
</section>
