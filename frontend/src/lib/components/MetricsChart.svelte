<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart, type ChartConfiguration, registerables } from 'chart.js';

	Chart.register(...registerables);

	export let data:   number[] = [];
	export let labels: string[] = [];
	export let label  = 'Value';
	export let color  = '#3b82f6';
	export let unit   = '';

	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	const config = (): ChartConfiguration => ({
		type: 'line',
		data: {
			labels,
			datasets: [{
				label,
				data,
				borderColor:     color,
				backgroundColor: color + '20',
				borderWidth:     2,
				pointRadius:     0,
				tension:         0.4,
				fill:            true
			}]
		},
		options: {
			responsive:          true,
			maintainAspectRatio: false,
			animation:           false,
			plugins: {
				legend: { display: false },
				tooltip: {
					callbacks: {
						label: (ctx) => `${ctx.parsed.y}${unit}`
					}
				}
			},
			scales: {
				x: {
					grid: { display: false },
					ticks: { color: '#9ca3af', font: { size: 11 } }
				},
				y: {
					beginAtZero: true,
					grid: { color: '#f3f4f6' },
					ticks: {
						color: '#9ca3af',
						font:  { size: 11 },
						callback: (v) => `${v}${unit}`
					}
				}
			}
		}
	});

	onMount(() => {
		chart = new Chart(canvas, config());
	});

	onDestroy(() => chart?.destroy());

	// Reactive update when data changes
	$: if (chart) {
		chart.data.labels   = labels;
		chart.data.datasets[0].data = data;
		chart.update('none');
	}
</script>

<canvas bind:this={canvas}></canvas>
