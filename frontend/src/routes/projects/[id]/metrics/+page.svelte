<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import type { ContainerMetrics, MetricsSnapshot } from '$types';
	import type { Chart as ChartInstance, ChartConfiguration, ChartItem } from 'chart.js';

	type LineChart = ChartInstance<'line', number[], string>;
	type ChartConstructor = new (
		item: ChartItem,
		config: ChartConfiguration<'line', number[], string>
	) => LineChart;
	type MetricKey = 'cpu' | 'memory';
	type MetricsPoint = {
		label: string;
		cpu: number;
		memory: number;
	};

	const metricOptions: Array<{ id: MetricKey; label: string; suffix: string; color: string }> = [
		{ id: 'cpu',    label: 'CPU',    suffix: '%',  color: '#111827' },
		{ id: 'memory', label: 'Memory', suffix: 'MB', color: '#059669' }
	];

	let snapshot: MetricsSnapshot | null = null;
	let history: MetricsPoint[] = [];
	let selectedMetric: MetricKey = 'cpu';
	let selectedService = '';
	let loading = true;
	let refreshing = false;
	let error = '';
	let chartCanvas: HTMLCanvasElement | null = null;
	let ChartCtor: ChartConstructor | null = null;
	let chart: LineChart | null = null;

	$: metricItems = snapshot?.items ?? [];
	$: services = metricItems.map((item) => item.service);
	$: primary = metricItems.find((item) => item.service === selectedService) ?? metricItems[0] ?? null;
	$: memoryPercent = primary && primary.memoryLimitMb > 0
		? Math.min((primary.memoryMb / primary.memoryLimitMb) * 100, 999)
		: 0;
	$: updateChart(history, selectedMetric, ChartCtor);

	onMount(() => {
		let cancelled = false;
		let interval: ReturnType<typeof setInterval> | undefined;

		void (async () => {
			const mod = await import('chart.js/auto');
			if (cancelled) return;
			ChartCtor = mod.default as ChartConstructor;
			await load();
			interval = setInterval(load, 5000);
		})();

		return () => {
			cancelled = true;
			if (interval) clearInterval(interval);
			chart?.destroy();
			chart = null;
		};
	});

	async function load() {
		refreshing = true;
		try {
			const result = await api.metrics.snapshot($page.params.id);
			const nextServices = result.items.map((item) => item.service);
			if (!selectedService || !nextServices.includes(selectedService)) {
				selectedService = nextServices[0] ?? '';
				resetChartHistory();
			}
			snapshot = result;
			error = '';
			const item = result.items.find((entry) => entry.service === selectedService) ?? result.items[0];
			if (item) appendHistory(result.collectedAt, item);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load metrics';
		} finally {
			loading = false;
			refreshing = false;
		}
	}

	function appendHistory(collectedAt: string, item: ContainerMetrics) {
		const label = collectedAt ? new Date(collectedAt).toLocaleTimeString() : new Date().toLocaleTimeString();
		history = [...history, { label, cpu: item.cpu, memory: item.memoryMb }].slice(-30);
	}

	function selectService(service: string) {
		if (service === selectedService) return;
		selectedService = service;
		resetChartHistory();
		const item = snapshot?.items.find((entry) => entry.service === selectedService);
		if (item) appendHistory(snapshot?.collectedAt ?? '', item);
	}

	function resetChartHistory() {
		history = [];
		chart?.destroy();
		chart = null;
	}

	function updateChart(points: MetricsPoint[], metric: MetricKey, ctor: ChartConstructor | null) {
		if (!ctor || !chartCanvas || points.length === 0) return;

		const option = metricOptions.find((item) => item.id === metric) ?? metricOptions[0];
		const values = points.map((point) => metric === 'cpu' ? point.cpu : point.memory);
		const labels = points.map((point) => point.label);

		if (!chart) {
			chart = new ctor(chartCanvas, chartConfig(labels, values, option));
			return;
		}

		chart.data.labels = labels;
		chart.data.datasets[0].label = option.label;
		chart.data.datasets[0].data = values;
		chart.data.datasets[0].borderColor = option.color;
		chart.data.datasets[0].backgroundColor = `${option.color}18`;
		chart.options.scales = chartConfig(labels, values, option).options?.scales;
		chart.update('none');
	}

	function chartConfig(
		labels: string[],
		values: number[],
		option: { label: string; suffix: string; color: string }
	): ChartConfiguration<'line', number[], string> {
		const maxValue = Math.max(10, ...values);
		return {
			type: 'line',
			data: {
				labels,
				datasets: [
					{
						label: option.label,
						data: values,
						borderColor: option.color,
						backgroundColor: `${option.color}18`,
						borderWidth: 2,
						fill: true,
						pointRadius: 0,
						tension: 0.35
					}
				]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				animation: false,
				plugins: {
					legend: { display: false },
					tooltip: {
						callbacks: {
							label: (context) => `${option.label}: ${(context.parsed.y ?? 0).toFixed(2)}${option.suffix}`
						}
					}
				},
				scales: {
					x: { grid: { display: false }, ticks: { maxTicksLimit: 6 } },
					y: {
						beginAtZero: true,
						suggestedMax: selectedMetric === 'cpu' ? 100 : Math.ceil(maxValue * 1.2),
						grid: { color: 'rgba(148, 163, 184, 0.16)' }
					}
				}
			}
		};
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
		<ActionButton
			variant="secondary"
			type="button"
			on:click={load}
			loading={refreshing}
			loadingLabel="Refreshing..."
		>
			Refresh
		</ActionButton>
	</div>

	{#if services.length > 1}
		<div class="flex flex-wrap items-center gap-2">
			{#each services as service}
				<button
					type="button"
					on:click={() => selectService(service)}
					class="rounded-md border px-3 py-1.5 text-sm font-medium transition-colors
						{selectedService === service
							? 'border-gray-900 bg-gray-900 text-white dark:border-white dark:bg-white dark:text-gray-950'
							: 'border-gray-200 bg-white text-gray-600 hover:text-gray-950 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300 dark:hover:text-white'}"
				>
					{service}
				</button>
			{/each}
		</div>
	{/if}

	{#if error}
		<div class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
			{error}
		</div>
	{/if}

	{#if loading && !primary}
		<div class="surface p-6">
			<p class="text-sm text-gray-500 dark:text-gray-400">Loading metrics...</p>
		</div>
	{:else if primary}
		<section class="surface overflow-hidden">
			<div class="grid divide-y divide-gray-100 dark:divide-gray-800 md:grid-cols-3 md:divide-x md:divide-y-0">
				<div class="p-5">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">CPU</p>
					<p class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primary.cpu.toFixed(2)}%</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{primary.service}</p>
				</div>
				<div class="p-5">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Memory</p>
					<p class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primary.memoryMb.toFixed(1)} MB</p>
					<div class="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
						<div class="h-full rounded-full bg-emerald-500" style={`width: ${Math.min(memoryPercent, 100)}%`}></div>
					</div>
				</div>
				<div class="p-5">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Uptime</p>
					<p class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primary.uptime}</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{primary.service}</p>
				</div>
			</div>
		</section>

		{#if metricItems.length > 1}
			<section class="surface overflow-hidden">
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
			</section>
		{/if}

		<section class="surface p-5">
			<div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Recent usage</h2>
				<div class="inline-flex rounded-md border border-gray-200 bg-gray-50 p-1 dark:border-gray-800 dark:bg-gray-950">
					{#each metricOptions as option}
						<button
							type="button"
							on:click={() => (selectedMetric = option.id)}
							class="rounded px-3 py-1.5 text-sm font-medium transition-colors
								{selectedMetric === option.id
									? 'bg-white text-gray-950 shadow-sm dark:bg-gray-800 dark:text-white'
									: 'text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white'}"
						>
							{option.label}
						</button>
					{/each}
				</div>
			</div>
			<div class="h-72">
				<canvas bind:this={chartCanvas}></canvas>
			</div>
		</section>
	{:else}
		<div class="surface p-6">
			<p class="text-sm text-gray-500 dark:text-gray-400">No metrics are available for this project yet.</p>
		</div>
	{/if}
</div>
