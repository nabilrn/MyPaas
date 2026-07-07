<script lang="ts">
	export let label = '';
	export let value = '';
	export let detail = '';
	export let percent = 0;
	export let tone: 'neutral' | 'success' | 'info' | 'warning' | 'danger' = 'neutral';
	export let className = '';

	const chartWidth = 180;
	const chartHeight = 76;
	const sampleShape = [0.1, 0.13, 0.11, 0.17, 0.16, 0.22, 0.2, 0.27, 0.25, 0.31, 0.29, 0.36, 0.34, 0.42, 0.4, 0.48];

	$: safePercent = Math.max(0, Math.min(100, Number.isFinite(percent) ? percent : 0));
	$: series = buildSeries(safePercent);
	$: points = series.map((level, index) => {
		const x = (index / Math.max(1, series.length - 1)) * chartWidth;
		const y = chartHeight - level * chartHeight;
		return `${x.toFixed(2)},${y.toFixed(2)}`;
	});
	$: linePath = `M ${points.join(' L ')}`;
	$: areaPath = `M 0 ${chartHeight} L ${points.join(' L ')} L ${chartWidth} ${chartHeight} Z`;
	$: toneClass = {
		neutral: {
			dot: 'bg-gray-400 dark:bg-gray-500',
			text: 'text-gray-600 dark:text-gray-300',
			stroke: 'stroke-gray-500 dark:stroke-gray-400',
			fill: 'fill-gray-400/10 dark:fill-gray-300/10'
		},
		success: {
			dot: 'bg-brand-500',
			text: 'text-brand-700 dark:text-brand-100',
			stroke: 'stroke-brand-500 dark:stroke-brand-500',
			fill: 'fill-brand-500/10 dark:fill-brand-500/15'
		},
		info: {
			dot: 'bg-sky-500',
			text: 'text-sky-700 dark:text-sky-300',
			stroke: 'stroke-sky-500 dark:stroke-sky-300',
			fill: 'fill-sky-500/10 dark:fill-sky-300/15'
		},
		warning: {
			dot: 'bg-amber-500',
			text: 'text-amber-700 dark:text-amber-200',
			stroke: 'stroke-amber-500 dark:stroke-amber-300',
			fill: 'fill-amber-500/10 dark:fill-amber-300/15'
		},
		danger: {
			dot: 'bg-red-500',
			text: 'text-red-700 dark:text-red-200',
			stroke: 'stroke-red-500 dark:stroke-red-300',
			fill: 'fill-red-500/10 dark:fill-red-300/15'
		}
	}[tone];

	function buildSeries(currentPercent: number) {
		const current = currentPercent / 100;
		return sampleShape.map((sample, index) => {
			if (index === sampleShape.length - 1) return current;
			if (current <= 0) return 0;
			const leadIn = 0.58 + sample * 0.62 + (index / sampleShape.length) * 0.18;
			return Math.max(0.03, Math.min(0.98, current * leadIn));
		});
	}
</script>

<article class={`min-w-0 p-4 ${className}`.trim()} aria-label={`${label} ${safePercent.toFixed(0)} percent`}>
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<div class="flex items-center gap-2">
				<span class={`h-1.5 w-1.5 rounded-full ${toneClass.dot}`}></span>
				<p class="metric-label truncate">{label}</p>
			</div>
			<p class="mt-1 truncate text-lg font-semibold tracking-tight text-gray-950 dark:text-white">{value}</p>
		</div>
		<p class={`font-mono text-xs font-semibold ${toneClass.text}`}>{safePercent.toFixed(0)}%</p>
	</div>

	<div class="mt-3 h-20 overflow-hidden rounded-md border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950">
		<svg class="h-full w-full" viewBox={`0 0 ${chartWidth} ${chartHeight}`} preserveAspectRatio="none" role="img" aria-hidden="true">
			<g class="stroke-gray-100 dark:stroke-gray-800" stroke-width="1">
				<line x1="0" x2={chartWidth} y1={chartHeight * 0.25} y2={chartHeight * 0.25} />
				<line x1="0" x2={chartWidth} y1={chartHeight * 0.5} y2={chartHeight * 0.5} />
				<line x1="0" x2={chartWidth} y1={chartHeight * 0.75} y2={chartHeight * 0.75} />
				<line x1={chartWidth * 0.25} x2={chartWidth * 0.25} y1="0" y2={chartHeight} />
				<line x1={chartWidth * 0.5} x2={chartWidth * 0.5} y1="0" y2={chartHeight} />
				<line x1={chartWidth * 0.75} x2={chartWidth * 0.75} y1="0" y2={chartHeight} />
			</g>
			<path d={areaPath} class={toneClass.fill} />
			<path d={linePath} fill="none" class={toneClass.stroke} stroke-width="2" vector-effect="non-scaling-stroke" />
		</svg>
	</div>

	<div class="mt-2 flex items-center justify-between gap-3 text-[11px] text-gray-500 dark:text-gray-400">
		<p class="truncate">{detail}</p>
		<span class="shrink-0 font-mono">0-100%</span>
	</div>
</article>
