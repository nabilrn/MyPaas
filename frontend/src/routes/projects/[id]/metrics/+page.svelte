<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import CapacityMetricChart from '$components/CapacityMetricChart.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import IconButton from '$components/IconButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import type { MetricsSnapshot } from '$types';

	let snapshot: MetricsSnapshot | null = null;
	let selectedService = '';
	let loading = true;
	let refreshing = false;
	let metricsInFlight = false;
	let error = '';

	$: metricItems = snapshot?.items ?? [];
	$: services = metricItems.map((item) => item.service);
	$: primary = metricItems.find((item) => item.service === selectedService) ?? metricItems[0] ?? null;
	$: memoryPercent = primary && primary.memoryLimitMb > 0
		? Math.min((primary.memoryMb / primary.memoryLimitMb) * 100, 999)
		: 0;
	$: cpuPercent = primary ? Math.min(primary.cpu, 100) : 0;
	$: runtimeSummary = primary
		? [
				{ label: 'Service', value: primary.service },
				{ label: 'Uptime', value: primary.uptime },
				{ label: 'Collected', value: snapshot?.collectedAt ? new Date(snapshot.collectedAt).toLocaleTimeString() : '-' }
			]
		: [];

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
		if (!background && !snapshot) {
			loading = true;
		}
		refreshing = true;
		try {
			const result = await api.metrics.snapshot($page.params.id);
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
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h1 class="text-lg font-semibold tracking-tight text-gray-950 dark:text-white">Metrics</h1>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
				{primary ? `Updated ${new Date(snapshot?.collectedAt ?? '').toLocaleTimeString()}` : 'Waiting for container metrics'}
			</p>
		</div>
		<IconButton label="Refresh metrics" variant="brand" loading={refreshing} on:click={() => void load()}>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M20 11a8.1 8.1 0 00-15.5-3M4 4v4h4m-4 5a8.1 8.1 0 0015.5 3M20 20v-4h-4" />
			</svg>
		</IconButton>
	</div>

	{#if error}
		{#if primary}
			<div class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
				Latest refresh failed. Showing the last collected metrics. {error}
			</div>
		{:else}
			<div class="surface overflow-hidden">
				<ErrorState title="Could not load metrics" message={error} on:retry={() => void load()} />
			</div>
		{/if}
	{/if}

	{#if loading && !primary}
		<div class="surface grid gap-0 overflow-hidden md:grid-cols-2">
			{#each [1, 2] as _}
				<div class="border-b border-gray-100 p-5 dark:border-gray-800 md:border-b-0 md:border-r">
					<div class="h-3 w-20 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
					<div class="mt-3 h-8 w-28 animate-pulse rounded bg-gray-200 dark:bg-gray-800"></div>
					<div class="mt-3 h-2 w-full animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
				</div>
			{/each}
		</div>
	{:else if primary}
		<SectionPanel
			title="Runtime usage"
			description="Current CPU and memory sample for the selected service."
			contentClass="p-0"
		>
			<svelte:fragment slot="actions">
				{#if services.length > 1}
					<label class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
						<span>Service</span>
						<select
							class="field h-8 min-w-36 !py-1 text-xs"
							value={selectedService}
							on:change={(event) => selectService((event.currentTarget as HTMLSelectElement).value)}
						>
							{#each services as service}
								<option value={service}>{service}</option>
							{/each}
						</select>
					</label>
				{/if}
			</svelte:fragment>

			<div class="grid gap-px bg-gray-100 dark:bg-gray-800 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_18rem]">
				<CapacityMetricChart
					label="CPU"
					value={`${primary.cpu.toFixed(2)}%`}
					detail="runtime sample"
					percent={cpuPercent}
					tone={cpuPercent >= 90 ? 'danger' : cpuPercent >= 75 ? 'warning' : 'info'}
					className="bg-white dark:bg-gray-900"
				/>
				<CapacityMetricChart
					label="Memory"
					value={`${primary.memoryMb.toFixed(1)} MB`}
					detail={`${primary.memoryLimitMb.toFixed(0)} MB limit`}
					percent={Math.min(memoryPercent, 100)}
					tone={memoryPercent >= 90 ? 'danger' : memoryPercent >= 75 ? 'warning' : 'success'}
					className="bg-white dark:bg-gray-900"
				/>
				<div class="bg-white p-4 dark:bg-gray-900">
					<p class="metric-label">Runtime context</p>
					<div class="mt-3 divide-y divide-gray-100 dark:divide-gray-800">
						{#each runtimeSummary as item}
							<div class="flex items-center justify-between gap-3 py-2 text-xs">
								<span class="text-gray-500 dark:text-gray-400">{item.label}</span>
								<span class="max-w-40 truncate text-right font-medium text-gray-950 dark:text-white">{item.value}</span>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</SectionPanel>

		{#if metricItems.length > 1}
			<SectionPanel
				title="Services"
				description="Container-level metrics reported for this project."
				contentClass="p-0"
			>
				<div class="grid divide-y divide-gray-100 dark:divide-gray-800">
					{#each metricItems as item}
						<button
							type="button"
							on:click={() => selectService(item.service)}
							class="grid gap-3 px-5 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-900 sm:grid-cols-[minmax(0,1fr)_7rem_9rem_7rem]"
						>
							<span class="truncate text-sm font-medium text-gray-950 dark:text-white">{item.service}</span>
							<span class="text-sm text-gray-600 dark:text-gray-300">{item.cpu.toFixed(2)}% CPU</span>
							<span class="text-sm text-gray-600 dark:text-gray-300">{item.memoryMb.toFixed(1)} / {item.memoryLimitMb.toFixed(0)} MB</span>
							<span class="text-sm text-gray-500 dark:text-gray-400">{item.uptime}</span>
						</button>
					{/each}
				</div>
			</SectionPanel>
		{/if}
	{:else if !error}
		<div class="surface overflow-hidden">
			<EmptyState
				title="No metrics yet."
				description="Metrics appear after the project has a running container or service."
				compact
			/>
		</div>
	{/if}
</div>
