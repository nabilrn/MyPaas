<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
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
		{ id: 'cpu',    label: 'CPU',    suffix: '%',  color: '#2563eb' },
		{ id: 'memory', label: 'Memory', suffix: 'MB', color: '#059669' }
	];

	let snapshot: MetricsSnapshot | null = null;
	let history: MetricsPoint[] = [];
	let selectedMetric: MetricKey = 'cpu';
	let loading = true;
	let refreshing = false;
	let error = '';
	let chartCanvas: HTMLCanvasElement | null = null;
	let ChartCtor: ChartConstructor | null = null;
	let chart: LineChart | null = null;

	$: primary = snapshot?.items[0] ?? null;
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
			snapshot = result;
			error = '';
			const item = result.items[0];
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
		history = [
			...history,
			{
				label,
				cpu: item.cpu,
				memory: item.memoryMb
			}
		].slice(-30);
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
		chart.data.datasets[0].backgroundColor = `${option.color}22`;
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
						backgroundColor: `${option.color}22`,
						borderWidth: 2,
						fill: true,
						pointRadius: 2,
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
					x: {
						grid: { display: false },
						ticks: { maxTicksLimit: 6 }
					},
					y: {
						beginAtZero: true,
						suggestedMax: selectedMetric === 'cpu' ? 100 : Math.ceil(maxValue * 1.2),
						grid: { color: 'rgba(148, 163, 184, 0.18)' }
					}
				}
			}
		};
	}
</script>

<svelte:head>
	<title>Metrics · MyPaas</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h1 class="text-xl font-bold text-gray-900 dark:text-white">Metrics</h1>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
				{primary ? `Updated ${new Date(snapshot?.collectedAt ?? '').toLocaleTimeString()}` : 'Waiting for container metrics'}
			</p>
		</div>
		<button
			type="button"
			on:click={load}
			disabled={refreshing}
			class="inline-flex items-center justify-center rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50
				   dark:border-gray-700 dark:text-gray-200 dark:hover:bg-gray-800"
		>
			{refreshing ? 'Refreshing...' : 'Refresh'}
		</button>
	</div>

	{#if error}
		<div class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
			{error}
		</div>
	{/if}

	{#if loading && !primary}
		<div class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
			<p class="text-sm text-gray-500 dark:text-gray-400">Loading metrics...</p>
		</div>
	{:else if primary}
		<div class="grid gap-4 md:grid-cols-3">
			<div class="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
				<p class="text-sm font-medium text-gray-500 dark:text-gray-400">CPU</p>
				<p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{primary.cpu.toFixed(2)}%</p>
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{primary.service}</p>
			</div>
			<div class="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
				<p class="text-sm font-medium text-gray-500 dark:text-gray-400">Memory</p>
				<p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">
					{primary.memoryMb.toFixed(1)} MB
				</p>
				<div class="mt-3 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
					<div class="h-full rounded-full bg-emerald-600" style={`width: ${Math.min(memoryPercent, 100)}%`}></div>
				</div>
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					{memoryPercent.toFixed(1)}% of {primary.memoryLimitMb.toFixed(0)} MB
				</p>
			</div>
			<div class="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
				<p class="text-sm font-medium text-gray-500 dark:text-gray-400">Uptime</p>
				<p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{primary.uptime}</p>
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Dockerfile service</p>
			</div>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
			<div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<h2 class="font-semibold text-gray-900 dark:text-white">Recent usage</h2>
				<div class="inline-flex rounded-lg border border-gray-200 p-1 dark:border-gray-800">
					{#each metricOptions as option}
						<button
							type="button"
							on:click={() => (selectedMetric = option.id)}
							class={`rounded-md px-3 py-1.5 text-sm font-medium ${
								selectedMetric === option.id
									? 'bg-brand-600 text-white'
									: 'text-gray-600 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800'
							}`}
						>
							{option.label}
						</button>
					{/each}
				</div>
			</div>
			<div class="h-64">
				<canvas bind:this={chartCanvas}></canvas>
			</div>
		</div>
	{:else}
		<div class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
			<p class="text-sm text-gray-500 dark:text-gray-400">
				No metrics are available for this project yet.
			</p>
		</div>
	{/if}
</div>
