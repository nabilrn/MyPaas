<script lang="ts">
	type Resource = 'memory' | 'cpu' | 'storage' | 'network' | 'neutral';

	export let label = '';
	export let value = '';
	export let detail = '';
	export let indicator = '';
	export let series: number[] = [];
	export let resource: Resource = 'neutral';
	export let maxValue: number | null = 100;
	export let rangeLabel = '0–100%';
	export let className = '';
	// Compatibility for current-value callsites. A single real sample is rendered
	// as a point only and is never expanded into synthetic history.
	export let percent: number | null = null;
	export let tone: 'neutral' | 'success' | 'info' | 'warning' | 'danger' = 'neutral';

	const chartWidth = 180;
	const chartHeight = 76;
	let inferredResource: Resource = 'neutral';
	const resourceClasses = {
		neutral: {
			dot: 'bg-gray-400 dark:bg-gray-500',
			stroke: 'stroke-gray-500 dark:stroke-gray-400',
			fill: 'fill-gray-400/10 dark:fill-gray-300/10'
		},
		memory: {
			dot: 'bg-emerald-500',
			stroke: 'stroke-emerald-500 dark:stroke-emerald-400',
			fill: 'fill-emerald-500/10 dark:fill-emerald-400/15'
		},
		cpu: {
			dot: 'bg-sky-500',
			stroke: 'stroke-sky-500 dark:stroke-sky-400',
			fill: 'fill-sky-500/10 dark:fill-sky-400/15'
		},
		storage: {
			dot: 'bg-amber-500',
			stroke: 'stroke-amber-500 dark:stroke-amber-400',
			fill: 'fill-amber-500/10 dark:fill-amber-400/15'
		},
		network: {
			dot: 'bg-violet-500',
			stroke: 'stroke-violet-500 dark:stroke-violet-400',
			fill: 'fill-violet-500/10 dark:fill-violet-400/15'
		}
	} as const;

	$: inferredResource = resource !== 'neutral'
		? resource
		: label.toLowerCase().includes('cpu')
			? 'cpu'
			: label.toLowerCase().includes('memory') || label.toLowerCase().includes('ram')
				? 'memory'
				: 'neutral';
	$: compatibilitySeries = percent !== null && Number.isFinite(percent) ? [Math.max(0, percent)] : [];
	$: cleanSeries = (series.length > 0 ? series : compatibilitySeries).filter((sample) => Number.isFinite(sample) && sample >= 0);
	$: effectiveIndicator = indicator || (percent !== null && Number.isFinite(percent) ? `${Math.max(0, percent).toFixed(0)}%` : '');
	$: effectiveMax = maxValue && maxValue > 0
		? maxValue
		: Math.max(1, ...cleanSeries) * 1.15;
	$: points = cleanSeries.map((sample, index) => {
		const x = cleanSeries.length === 1 ? chartWidth - 4 : (index / Math.max(1, cleanSeries.length - 1)) * chartWidth;
		const level = Math.max(0, Math.min(1, sample / effectiveMax));
		const y = chartHeight - level * chartHeight;
		return { x, y, encoded: `${x.toFixed(2)},${y.toFixed(2)}` };
	});
	$: linePath = points.length >= 2 ? `M ${points.map((point) => point.encoded).join(' L ')}` : '';
	$: areaPath = points.length >= 2 ? `M 0 ${chartHeight} L ${points.map((point) => point.encoded).join(' L ')} L ${chartWidth} ${chartHeight} Z` : '';
	$: resourceClass = resourceClasses[inferredResource] ?? resourceClasses.neutral;
	$: currentPoint = points.length === 1 ? points[0] : null;
	$: currentOnly = series.length === 0 && compatibilitySeries.length === 1;
	$: compatibilityTone = tone;
</script>

<article class={`min-w-0 p-4 ${className}`.trim()} aria-label={`${label}: ${value}`} data-legacy-tone={compatibilityTone}>
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<div class="flex items-center gap-2">
				<span class={`h-1.5 w-1.5 rounded-full ${resourceClass.dot}`}></span>
				<p class="metric-label truncate">{label}</p>
			</div>
			<p class="metric-value mt-1 truncate text-lg font-semibold tracking-tight text-gray-950 dark:text-white">{value}</p>
		</div>
		{#if effectiveIndicator}
			<p class="metric-value shrink-0 text-xs font-semibold text-gray-500 dark:text-gray-400">{effectiveIndicator}</p>
		{/if}
	</div>

	<div class="mt-3 h-20 overflow-hidden rounded-md border border-gray-200 bg-white dark:border-gray-800 dark:bg-neutral-950">
		<svg class="h-full w-full" viewBox={`0 0 ${chartWidth} ${chartHeight}`} preserveAspectRatio="none" role="img" aria-hidden="true">
			<g class="stroke-gray-100 dark:stroke-neutral-800" stroke-width="1">
				<line x1="0" x2={chartWidth} y1={chartHeight * 0.25} y2={chartHeight * 0.25} />
				<line x1="0" x2={chartWidth} y1={chartHeight * 0.5} y2={chartHeight * 0.5} />
				<line x1="0" x2={chartWidth} y1={chartHeight * 0.75} y2={chartHeight * 0.75} />
				<line x1={chartWidth * 0.25} x2={chartWidth * 0.25} y1="0" y2={chartHeight} />
				<line x1={chartWidth * 0.5} x2={chartWidth * 0.5} y1="0" y2={chartHeight} />
				<line x1={chartWidth * 0.75} x2={chartWidth * 0.75} y1="0" y2={chartHeight} />
			</g>
			{#if areaPath}<path d={areaPath} class={resourceClass.fill} />{/if}
			{#if linePath}<path d={linePath} fill="none" class={resourceClass.stroke} stroke-width="2" vector-effect="non-scaling-stroke" />{/if}
			{#if currentPoint}<circle cx={currentPoint.x} cy={currentPoint.y} r="3" class={`${resourceClass.stroke} ${resourceClass.fill}`} stroke-width="1.5" />{/if}
		</svg>
	</div>

	<div class="mt-2 flex items-center justify-between gap-3 text-[11px] text-gray-500 dark:text-gray-400">
		<p class="truncate">{detail}</p>
		<span class="shrink-0 font-mono">{currentOnly ? 'current' : cleanSeries.length >= 2 ? rangeLabel : 'collecting'}</span>
	</div>
</article>
