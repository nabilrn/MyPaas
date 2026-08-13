<script lang="ts">
	import { onMount } from 'svelte';
	import { Activity, AlertTriangle, Globe, LoaderCircle, Radio } from '@lucide/svelte';
	import { api } from '$api';
	import CapacityMetricChart from '$components/CapacityMetricChart.svelte';
	import CloudflareChart from '$components/CloudflareChart.svelte';
	import CloudflareSetup from '$components/CloudflareSetup.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { projectStreamConnection, projectStreamMetrics } from '$stores/project-stream';
	import { appendProjectMetricHistory, type ProjectMetricHistory } from '$lib/utils/project-metric-history';
	import type { CloudflareAnalytics, Project } from '$types';

	export let project: Project;

	let analytics: CloudflareAnalytics | null = null;
	let cloudflareConfigured: boolean | null = null;
	let analyticsLoading = true;
	let analyticsError = '';
	let selectedService = '';
	let metricHistory: ProjectMetricHistory = {};
	let lastHistorySample = '';

	$: metricItems = $projectStreamMetrics?.items ?? [];
	$: services = metricItems.map((item) => item.service);
	$: effectiveSelectedService = selectedService && services.includes(selectedService) ? selectedService : (services[0] ?? '');
	$: primary = metricItems.find((item) => item.service === effectiveSelectedService) ?? metricItems[0] ?? null;
	$: currentSampleKey = $projectStreamMetrics?.collectedAt ?? '';
	$: if (currentSampleKey && currentSampleKey !== lastHistorySample) {
		metricHistory = appendProjectMetricHistory(metricHistory, metricItems);
		lastHistorySample = currentSampleKey;
	}
	$: primaryHistory = primary ? (metricHistory[primary.service] ?? { cpu: [], memoryPercent: [] }) : { cpu: [], memoryPercent: [] };
	$: memoryPercent = primary && primary.memoryLimitMb > 0 ? Math.min((primary.memoryMb / primary.memoryLimitMb) * 100, 100) : 0;
	$: cpuPercent = primary ? Math.min(primary.cpu, 100) : 0;
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

	function selectService(service: string) {
		selectedService = service;
	}
</script>

<section class="space-y-4" aria-label="Runtime and edge observability">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h2 class="text-base font-semibold text-gray-950 dark:text-white">Runtime & edge traffic</h2>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Live runtime samples stream independently from slower edge analytics.</p>
		</div>
		{#if project.deployMode !== 'static'}
			<div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400" aria-live="polite">
				{#if $projectStreamConnection === 'connecting'}
					<LoaderCircle class="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" /> Connecting to telemetry…
				{:else if $projectStreamConnection === 'reconnecting'}
					<LoaderCircle class="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" /> Reconnecting{sampleLabel ? ` · last sample ${sampleLabel}` : '…'}
				{:else}
					<Radio class="h-3.5 w-3.5" /> Live{sampleLabel ? ` · ${sampleLabel}` : ''}
				{/if}
			</div>
		{/if}
	</div>

	{#if analyticsLoading && !analytics}
		<div class="surface flex min-h-44 items-center justify-center gap-2 text-sm text-gray-500 dark:text-gray-400">
			<LoaderCircle class="h-4 w-4 animate-spin motion-reduce:animate-none" /> Loading edge analytics…
		</div>
	{:else if cloudflareConfigured === false}
		<CloudflareSetup on:success={() => { cloudflareConfigured = true; void loadAnalytics(); }} />
	{:else if analytics}
		<SectionPanel title="Edge traffic" description="Cloudflare analytics over the last 24 hours. Refreshed on a slow path." contentClass="p-0">
			<div class="grid grid-cols-3 divide-x divide-gray-100 border-b border-gray-100 dark:divide-neutral-800 dark:border-neutral-800">
				<div class="p-4 sm:p-5">
					<div class="flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400"><Activity class="h-3.5 w-3.5" /> Requests</div>
					<p class="metric-value mt-2 text-2xl font-semibold text-gray-950 dark:text-white">{analytics.total_requests.toLocaleString()}</p>
				</div>
				<div class="p-4 sm:p-5">
					<div class="flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400"><Globe class="h-3.5 w-3.5" /> Bandwidth</div>
					<p class="metric-value mt-2 text-2xl font-semibold text-gray-950 dark:text-white">{(analytics.bandwidth / (1024 * 1024)).toFixed(2)} <span class="text-sm font-medium text-gray-500">MB</span></p>
				</div>
				<div class="p-4 sm:p-5">
					<div class="flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400"><AlertTriangle class="h-3.5 w-3.5 {analytics.errors > 0 ? 'text-amber-500' : ''}" /> Edge errors</div>
					<p class="metric-value mt-2 text-2xl font-semibold text-gray-950 dark:text-white">{analytics.errors.toLocaleString()}</p>
				</div>
			</div>
			<div class="min-w-0 p-4 sm:p-5">
				{#if analytics.timeseries?.length > 0}
					<CloudflareChart data={analytics.timeseries} />
				{:else}
					<div class="flex h-48 items-center justify-center rounded-md border border-dashed border-gray-200 text-sm text-gray-500 dark:border-neutral-800 dark:text-gray-400">Not enough edge traffic yet to draw a 24-hour chart.</div>
				{/if}
			</div>
		</SectionPanel>
	{:else if analyticsError}
		<div class="surface px-5 py-4 text-sm text-gray-500 dark:text-gray-400">Edge analytics unavailable. {analyticsError}</div>
	{/if}

	{#if project.deployMode !== 'static'}
		{#if primary}
			<SectionPanel title="Live runtime" description="CPU and memory arrive over the shared project SSE stream." contentClass="p-0">
				<svelte:fragment slot="actions">
					{#if services.length > 1}
						<label class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
							<span>Service</span>
							<select class="field h-8 min-w-36 !py-1 text-xs" value={effectiveSelectedService} on:change={(event) => selectService((event.currentTarget as HTMLSelectElement).value)}>
								{#each services as service}<option value={service}>{service}</option>{/each}
							</select>
						</label>
					{/if}
				</svelte:fragment>
				<div class="grid gap-px bg-gray-100 dark:bg-neutral-800 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_18rem]">
					<CapacityMetricChart label="CPU" value={`${primary.cpu.toFixed(2)}%`} detail="live rolling samples" percent={cpuPercent} series={primaryHistory.cpu} resource="cpu" className="bg-white dark:bg-neutral-900" />
					<CapacityMetricChart label="Memory" value={`${primary.memoryMb.toFixed(1)} MB`} detail={`${primary.memoryLimitMb.toFixed(0)} MB limit`} percent={memoryPercent} series={primaryHistory.memoryPercent} resource="memory" className="bg-white dark:bg-neutral-900" />
					<div class="bg-white p-5 dark:bg-neutral-900">
						<p class="metric-label">Runtime context</p>
						<div class="mt-3 divide-y divide-gray-100 dark:divide-neutral-800">
							<div class="flex justify-between gap-3 py-2.5 text-sm"><span class="text-gray-500 dark:text-gray-400">Service</span><span class="font-medium text-gray-950 dark:text-white">{primary.service}</span></div>
							<div class="flex justify-between gap-3 py-2.5 text-sm"><span class="text-gray-500 dark:text-gray-400">Uptime</span><span class="metric-value font-medium text-gray-950 dark:text-white">{primary.uptime}</span></div>
							<div class="flex justify-between gap-3 py-2.5 text-sm"><span class="text-gray-500 dark:text-gray-400">Transport</span><span class="font-medium text-gray-950 dark:text-white">SSE</span></div>
							<div class="flex justify-between gap-3 py-2.5 text-sm"><span class="text-gray-500 dark:text-gray-400">Telemetry</span><span class="font-medium text-gray-950 dark:text-white">statd preferred</span></div>
						</div>
					</div>
				</div>
			</SectionPanel>
		{:else if $projectStreamConnection === 'connecting' || $projectStreamConnection === 'reconnecting'}
			<div class="surface flex min-h-40 items-center justify-center gap-2 text-sm text-gray-500 dark:text-gray-400">
				<LoaderCircle class="h-4 w-4 animate-spin motion-reduce:animate-none" /> {$projectStreamConnection === 'reconnecting' ? 'Reconnecting to runtime telemetry…' : 'Connecting to runtime telemetry…'}
			</div>
		{:else}
			<div class="surface overflow-hidden">
				<EmptyState title="No active runtime metrics." description="CPU and memory samples appear only while a container runtime is active." compact />
			</div>
		{/if}
	{/if}
</section>
