<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart, type ChartConfiguration, registerables } from 'chart.js';
	import type { TimeseriesDataPoint } from '$types';

	Chart.register(...registerables);

	export let data: TimeseriesDataPoint[] = [];
	export let compact = false;

	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	$: labels = data.map((d) => {
		const date = new Date(d.timestamp);
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	});
	$: requestsData = data.map((d) => d.requests);
	$: bandwidthData = data.map((d) => Number((d.bandwidth / (1024 * 1024)).toFixed(2)));

	const config = (): ChartConfiguration => ({
		type: 'line',
		data: {
			labels,
			datasets: [
				{
					label: 'Requests',
					data: requestsData,
					borderColor: '#3b82f6',
					backgroundColor: '#3b82f620',
					borderWidth: compact ? 1.5 : 2,
					pointRadius: 0,
					pointHoverRadius: compact ? 2 : 4,
					tension: 0.4,
					fill: !compact,
					yAxisID: 'y'
				},
				{
					label: 'Bandwidth (MB)',
					data: bandwidthData,
					borderColor: '#10b981',
					backgroundColor: '#10b98120',
					borderWidth: compact ? 1.5 : 2,
					pointRadius: 0,
					pointHoverRadius: compact ? 2 : 4,
					tension: 0.4,
					fill: !compact,
					yAxisID: 'y1'
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
				legend: {
					display: !compact,
					position: 'top',
					labels: {
						usePointStyle: true,
						boxWidth: 6
					}
				},
				tooltip: {
					enabled: !compact,
					backgroundColor: 'rgba(17, 24, 39, 0.9)',
					titleColor: '#fff',
					bodyColor: '#e5e7eb',
					borderColor: 'rgba(255,255,255,0.1)',
					borderWidth: 1,
					padding: 10,
					boxPadding: 4,
					usePointStyle: true
				}
			},
			scales: {
				x: {
					display: !compact,
					grid: { display: true, color: 'rgba(156, 163, 175, 0.05)', drawTicks: false },
					border: { display: false },
					ticks: { color: '#9ca3af', font: { size: 11 }, maxTicksLimit: 8 }
				},
				y: {
					type: 'linear',
					display: !compact,
					position: 'left',
					beginAtZero: true,
					grid: { color: 'rgba(156, 163, 175, 0.05)', drawTicks: false },
					border: { display: false },
					ticks: {
						color: '#9ca3af',
						font: { size: 11 }
					}
				},
				y1: {
					type: 'linear',
					display: !compact,
					position: 'right',
					beginAtZero: true,
					grid: { display: false },
					border: { display: false },
					ticks: {
						color: '#9ca3af',
						font: { size: 11 },
						callback: (v) => `${v} MB`
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
		chart.data.datasets[0].data = requestsData;
		chart.data.datasets[1].data = bandwidthData;
		chart.update(compact ? 'none' : undefined);
	}
</script>

<div class={`${compact ? 'h-16' : 'h-64'} relative w-full`}>
	<canvas bind:this={canvas}></canvas>
</div>
