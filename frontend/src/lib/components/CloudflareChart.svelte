<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart, type ChartConfiguration, registerables } from 'chart.js';
	import type { TimeseriesDataPoint } from '$types';

	Chart.register(...registerables);

	type EdgeMetric = 'requests' | 'bandwidth';

	export let data: TimeseriesDataPoint[] = [];
	export let metric: EdgeMetric = 'requests';
	export let compact = false;

	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	$: labels = data.map((d) => {
		const date = new Date(d.timestamp);
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	});
	$: values = data.map((point) => metric === 'requests'
		? point.requests
		: Number((point.bandwidth / (1024 * 1024)).toFixed(3)));
	$: metricLabel = metric === 'requests' ? 'Requests' : 'Bandwidth (MB)';
	$: barColor = metric === 'requests' ? 'rgba(59, 130, 246, 0.72)' : 'rgba(16, 185, 129, 0.72)';
	$: hoverColor = metric === 'requests' ? 'rgba(37, 99, 235, 0.9)' : 'rgba(5, 150, 105, 0.9)';
	$: domain = adaptiveDomain(values, metric);

	function adaptiveDomain(series: number[], selectedMetric: EdgeMetric) {
		const clean = series.filter((value) => Number.isFinite(value));
		if (clean.length === 0) return { min: 0, max: 1 };

		const observedMin = Math.min(...clean);
		const observedMax = Math.max(...clean);
		const span = observedMax - observedMin;
		const fallbackPadding = selectedMetric === 'requests'
			? Math.max(1, Math.abs(observedMax) * 0.08)
			: Math.max(0.001, Math.abs(observedMax) * 0.08);
		const padding = span > 0 ? span * 0.18 : fallbackPadding;
		const min = Math.max(0, observedMin - padding);
		let max = observedMax + padding;
		if (max <= min) max = min + (selectedMetric === 'requests' ? 1 : 0.001);
		return { min, max };
	}

	const config = (): ChartConfiguration<'bar'> => ({
		type: 'bar',
		data: {
			labels,
			datasets: [
				{
					label: metricLabel,
					data: values,
					backgroundColor: barColor,
					hoverBackgroundColor: hoverColor,
					borderWidth: 0,
					borderRadius: compact ? 2 : 3,
					borderSkipped: false,
					barPercentage: compact ? 0.78 : 0.72,
					categoryPercentage: compact ? 0.92 : 0.86,
					minBarLength: 2
				}
			]
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			animation: compact ? false : undefined,
			interaction: {
				mode: 'index',
				intersect: false
			},
			plugins: {
				legend: { display: false },
				tooltip: {
					enabled: true,
					backgroundColor: 'rgba(17, 24, 39, 0.9)',
					titleColor: '#fff',
					bodyColor: '#e5e7eb',
					borderColor: 'rgba(255,255,255,0.1)',
					borderWidth: 1,
					padding: compact ? 8 : 10,
					boxPadding: 4,
					usePointStyle: true
				}
			},
			scales: {
				x: {
					display: !compact,
					grid: { display: false, drawTicks: false },
					border: { display: false },
					ticks: { color: '#9ca3af', font: { size: 11 }, maxTicksLimit: 8 }
				},
				y: {
					type: 'linear',
					display: !compact,
					position: 'left',
					beginAtZero: false,
					min: domain.min,
					max: domain.max,
					grid: { color: 'rgba(156, 163, 175, 0.08)', drawTicks: false },
					border: { display: false },
					ticks: {
						color: '#9ca3af',
						font: { size: 11 },
						callback: (value) => metric === 'bandwidth' ? `${value} MB` : `${value}`
					}
				}
			}
		}
	});

	onMount(() => {
		chart = new Chart(canvas, config());
	});

	onDestroy(() => chart?.destroy());

	$: if (chart && data.length > 0) {
		chart.data.labels = labels;
		chart.data.datasets[0].label = metricLabel;
		chart.data.datasets[0].data = values;
		chart.data.datasets[0].backgroundColor = barColor;
		chart.data.datasets[0].hoverBackgroundColor = hoverColor;
		if (chart.options.scales?.y) {
			chart.options.scales.y.min = domain.min;
			chart.options.scales.y.max = domain.max;
		}
		chart.update(compact ? 'none' : undefined);
	}
</script>

<div class={`${compact ? 'h-24' : 'h-64'} relative w-full`}>
	<canvas bind:this={canvas}></canvas>
</div>
