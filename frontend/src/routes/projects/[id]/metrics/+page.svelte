<script lang="ts">
	import { Activity, AlertTriangle, Globe, RefreshCw } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import CapacityMetricChart from '$components/CapacityMetricChart.svelte';
	import CloudflareChart from '$components/CloudflareChart.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import { appendProjectMetricHistory, type ProjectMetricHistory } from '$lib/utils/project-metric-history';
	import type { MetricsSnapshot } from '$types';
	import CloudflareSetup from './CloudflareSetup.svelte';

	let snapshot: MetricsSnapshot | null = null;
	let cloudflareConfigured: boolean | null = null;
	let selectedService = '';
	let loading = true;
	let refreshing = false;
	let metricsInFlight = false;
	let error = '';
	let metricHistory: ProjectMetricHistory = {};

	$: metricItems = snapshot?.items ?? [];
	$: services = metricItems.map((item) => item.service);
	$: primary = metricItems.find((item) => item.service === selectedService) ?? metricItems[0] ?? null;
	$: primaryHistory = primary ? (metricHistory[primary.service] ?? { cpu: [], memoryPercent: [] }) : { cpu: [], memoryPercent: [] };
	$: memoryPercent = primary && primary.memoryLimitMb > 0
		? Math.min((primary.memoryMb / primary.memoryLimitMb) * 100, 999)
		: 0;
	$: cpuPercent = primary ? Math.min(primary.cpu, 100) : 0;
	$: runtimeSummary = primary
		? [
				{ label: 'Service', value: primary.service },
				{ label: 'Uptime', value: primary.uptime },
				{ label: 'Collected', value: snapshot?.collectedAt ? new Date(snapshot.collectedAt).toLocaleTimeString() : '-' },
				{ label: 'Telemetry', value: 'statd preferred' }
			]
		: [];
	$: updatedLabel = primary && snapshot?.collectedAt
		? `Updated ${new Date(snapshot.collectedAt).toLocaleTimeString()}`
		: 'Waiting for container metrics';

	onMount(() => {
		let interval: ReturnType<typeof setInterval> | undefined;
		void load();
		interval = setInterval(() => void load(true), 5000);
		return () => {
			if (interval) clearInterval(interval);
		};
	});

	async function load(background = false) {
		if (metricsInFlight) return;
		metricsInFlight = true;

		if (cloudflareConfigured === null) {
			try {
				const settings = await api.admin.getSettings();
				cloudflareConfigured = !!settings.cloudflare_configured;
			} catch {
				// Cloudflare analytics is optional; runtime metrics still load normally.
			}
		}

		if (!background && !snapshot) loading = true;
		refreshing = true;
		try {
			const result = await api.metrics.snapshot($page.params.id ?? '');
			metricHistory = appendProjectMetricHistory(metricHistory, result.items);
			const nextServices = result.items.map((item) => item.service);
			if (!selectedService || !nextServices.includes(selectedService)) {
				selectedService = nextServices[0] ?? '';
			}
			snapshot = result;
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load metrics';
		} finally {
			loading = false;
			refreshing = false;
			metricsInFlight = false;
		}
	}

	function selectService(service: string) {
		if (service === selectedService) return;
		selectedService = service;
	}
</script>

<svelte:head>
	<title>Metrics · MyPaas</title>
</svelte:head>

<div class="space-y-4">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<p class="text-sm text-gray-700 dark:text-gray-300">Service-level runtime diagnostics and optional edge traffic analytics.</p>
			<p class="mt-1 text-xs text-gray-500 dark:text-gray-400" aria-live="polite">{updatedLabel} · mypaas-statd preferred, container-engine fallback</p>
		</div>
		<ActionButton variant="secondary" size="sm" loading={refreshing} loadingLabel="Refreshing" on:click={() => void load()}>
			<RefreshCw slot="icon" class="h-4 w-4" />
			Refresh
		</ActionButton>
	</div>

	{#if error}
		{#if primary}
			<div class="alert-warning">
				<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
				<p>Latest refresh failed. Showing the last collected metrics. {error}</p>
			</div>
		{:else}
			<div class="surface overflow-hidden">
				<ErrorState title="Could not load metrics" message={error} on:retry={() => void load()} />
			</div>
		{/if}
	{/if}

	{#if cloudflareConfigured === false}
		<CloudflareSetup on:success={() => { cloudflareConfigured = true; load(); }} />
	{:else if snapshot?.analytics}
		<SectionPanel title="Cloudflare analytics" description="Edge traffic over the last 24 hours." contentClass="p-0">
			<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 lg:grid-cols-[18rem_minmax(0,1fr)] lg:divide-x lg:divide-y-0">
				<div class="grid grid-cols-3 divide-x divide-gray-100 dark:divide-neutral-800 lg:grid-cols-1 lg:divide-x-0 lg:divide-y">
					<div class="p-4">
						<div class="flex items-center gap-2 text-sm font-medium text-gray-500 dark:text-gray-400">
							<Activity class="h-4 w-4" aria-hidden="true" />
							<span>Requests</span>
						</div>
						<p class="metric-value mt-1 text-xl font-semibold tracking-tight text-gray-950 dark:text-white">{snapshot.analytics.total_requests.toLocaleString()}</p>
					</div>
					<div class="p-4">
						<div class="flex items-center gap-2 text-sm font-medium text-gray-500 dark:text-gray-400">
							<Globe class="h-4 w-4" aria-hidden="true" />
							<span>Bandwidth</span>
						</div>
						<p class="metric-value mt-1 text-xl font-semibold tracking-tight text-gray-950 dark:text-white">{(snapshot.analytics.bandwidth / (1024 * 1024)).toFixed(2)} MB</p>
					</div>
					<div class="p-4">
						<div class="flex items-center gap-2 text-sm font-medium text-gray-500 dark:text-gray-400">
							<AlertTriangle class="h-4 w-4 {snapshot.analytics.errors > 0 ? 'text-amber-500' : ''}" aria-hidden="true" />
							<span>Edge errors</span>
						</div>
						<p class="metric-value mt-1 text-xl font-semibold tracking-tight text-gray-950 dark:text-white">{snapshot.analytics.errors.toLocaleString()}</p>
					</div>
				</div>
				<div class="min-w-0 p-4">
					{#if snapshot.analytics.timeseries?.length > 0}
						<CloudflareChart data={snapshot.analytics.timeseries} />
					{:else}
						<div class="flex h-56 items-center justify-center border border-dashed border-gray-200 text-sm text-gray-500 dark:border-neutral-800 dark:text-gray-400">
							Not enough data to display timeseries
						</div>
					{/if}
				</div>
			</div>
		</SectionPanel>
	{/if}

	{#if loading && !primary}
		<div class="surface grid overflow-hidden md:grid-cols-2">
			{#each [1, 2] as _}
				<div class="border-b border-gray-100 p-4 last:border-b-0 dark:border-neutral-800 md:border-b-0 md:border-r">
					<div class="h-3 w-20 animate-pulse rounded bg-gray-100 dark:bg-neutral-800"></div>
					<div class="mt-3 h-8 w-28 animate-pulse rounded bg-gray-200 dark:bg-neutral-800"></div>
					<div class="mt-3 h-20 animate-pulse rounded-md bg-gray-100 dark:bg-neutral-800"></div>
				</div>
			{/each}
		</div>
	{:else if primary}
		<SectionPanel title="Service diagnostics" description="Current CPU and memory sample for the selected runtime service." contentClass="p-0">
			<svelte:fragment slot="actions">
				{#if services.length > 1}
					<label class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
						<span>Service</span>
						<select class="field h-9 min-w-40 !py-1.5 text-sm" value={selectedService} on:change={(event) => selectService((event.currentTarget as HTMLSelectElement).value)}>
							{#each services as service}<option value={service}>{service}</option>{/each}
						</select>
					</label>
				{/if}
			</svelte:fragment>

			<div class="grid gap-px bg-gray-100 dark:bg-neutral-800 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_20rem]">
				<CapacityMetricChart label="CPU" value={`${primary.cpu.toFixed(2)}%`} detail="rolling samples · current runtime" percent={cpuPercent} series={primaryHistory.cpu} resource="cpu" className="bg-white dark:bg-neutral-900" />
				<CapacityMetricChart label="Memory" value={`${primary.memoryMb.toFixed(1)} MB`} detail={`${primary.memoryLimitMb.toFixed(0)} MB limit · rolling usage`} percent={Math.min(memoryPercent, 100)} series={primaryHistory.memoryPercent} resource="memory" className="bg-white dark:bg-neutral-900" />
				<div class="bg-white p-4 dark:bg-neutral-900">
					<p class="metric-label">Runtime context</p>
					<div class="mt-2 divide-y divide-gray-100 dark:divide-neutral-800">
						{#each runtimeSummary as item}
							<div class="flex items-center justify-between gap-3 py-2.5 text-sm">
								<span class="text-gray-500 dark:text-gray-400">{item.label}</span>
								<span class="max-w-40 truncate text-right font-medium text-gray-950 dark:text-white">{item.value}</span>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</SectionPanel>

		{#if metricItems.length > 1}
			<SectionPanel title="Services" description="Container-level metrics reported for this project." contentClass="p-0">
				<div class="divide-y divide-gray-100 dark:divide-neutral-800">
					{#each metricItems as item}
						<button
							type="button"
							on:click={() => selectService(item.service)}
							class="app-focus grid w-full gap-2 px-4 py-2.5 text-left transition-colors hover:bg-gray-50 dark:hover:bg-neutral-900 sm:grid-cols-[minmax(0,1fr)_8rem_11rem_7rem] sm:items-center"
						>
							<span class="truncate text-sm font-medium text-gray-950 dark:text-white">{item.service}</span>
							<span class="metric-value text-sm text-gray-600 dark:text-gray-300">{item.cpu.toFixed(2)}% CPU</span>
							<span class="metric-value text-sm text-gray-600 dark:text-gray-300">{item.memoryMb.toFixed(1)} / {item.memoryLimitMb.toFixed(0)} MB</span>
							<span class="metric-value text-sm text-gray-500 dark:text-gray-400">{item.uptime}</span>
						</button>
					{/each}
				</div>
			</SectionPanel>
		{/if}
	{:else if !error}
		<div class="surface overflow-hidden">
			<EmptyState title="No active runtime metrics." description="Static projects and stopped runtimes do not expose container CPU or memory samples. Edge analytics remains available when Cloudflare is configured." compact />
		</div>
	{/if}
</div>
