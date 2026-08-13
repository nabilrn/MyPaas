<script lang="ts">
	import { deriveAdaptiveMetricDomain } from '$lib/utils/host-telemetry';

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
	export let percent: number | null = null;
	export let tone: 'neutral' | 'success' | 'info' | 'warning' | 'danger' = 'neutral';

	const chartWidth = 180;
	const chartHeight = 76;
	let inferredResource: Resource = 'neutral';
	const resourceClasses = {
		neutral: {
			dot: 'bg-gray-400 dark:bg-gray-500',
			stroke: 'stroke-gray-500/55 dark:stroke-gray-300/50',
			fill: 'fill-gray-400/[0.025] dark:fill-gray-300/[0.035]'
		},
		memory: {
			dot: 'bg-emerald-400/80 dark:bg-emerald-300/75',
			stroke: 'stroke-emerald-500/55 dark:stroke-emerald-300/50',
			fill: 'fill-emerald-400/[0.025] dark:fill-emerald-300/[0.035]'
		},
		cpu: {
			dot: 'bg-sky-400/80 dark:bg-sky-300/75',
			stroke: 'stroke-sky-500/55 dark:stroke-sky-300/50',
			fill: 'fill-sky-400/[0.025] dark:fill-sky-300/[0.035]'
		},
		storage: {
			dot: 'bg-amber-400/80 dark:bg-amber-300/75',
			stroke: 'stroke-amber-500/55 dark:stroke-amber-300/50',
			fill: 'fill-amber-400/[0.025] dark:fill-amber-300/[0.035]'
		},
		network: {
			dot: 'bg-violet-400/80 dark:bg-violet-300/75',
			stroke: 'stroke-violet-500/55 dark:stroke-violet-300/50',
			fill: 'fill-violet-400/[0.025] dark:fill-violet-300/[0.035]'
		}
	} as const;

	function formatDomainValue(value: number, span: number) {
		return span < 10 ? value.toFixed(1) : Math.round(value).toString();
	}

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
	$: rollingHistory = series.length >= 2 && cleanSeries.length >= 2;
	$: domain = deriveAdaptiveMetricDomain(cleanSeries, maxValue);
	$: domainMin = domain.min;
	$: domainMax = domain.max;
	$: domainSpan = Math.max(0.0001, domainMax - domainMin);
	$: points = cleanSeries.map((sample, index) => {
		const x = cleanSeries.length === 1 ? chartWidth - 4 : (index / Math.max(1, cleanSeries.length - 1)) * chartWidth;
		const level = Math.max(0, Math.min(1, (sample - domainMin) / domainSpan));
		const y = chartHeight - level * chartHeight;
		return { x, y, encoded: `${x.toFixed(2)},${y.toFixed(2)}` };
	});
	$: linePath = points.length >= 2 ? `M ${points.map((point) => point.encoded).join(' L ')}` : '';
	$: areaPath = points.length >= 2 ? `M 0 ${chartHeight} L ${points.map((point) => point.encoded).join(' L ')} L ${chartWidth} ${chartHeight} Z` : '';
	$: resourceClass = resourceClasses[inferredResource] ?? resourceClasses.neutral;
	$: currentPoint = points.length === 1 ? points[0] : null;
	$: currentOnly = series.length === 0 && compatibilitySeries.length === 1;
	$: telemetryUnavailable = cleanSeries.length === 0 && /unavailable/i.test(`${value} ${detail}`);
	$: chartMessage = cleanSeries.length === 0 ? (telemetryUnavailable ? 'Telemetry unavailable' : 'Waiting for sample') : '';
	$: effectiveRangeLabel = currentOnly
		? 'current'
		: rollingHistory && maxValue && maxValue > 0
			? `${formatDomainValue(domainMin, domainSpan)}–${formatDomainValue(domainMax, domainSpan)}%`
			: rollingHistory
				? rangeLabel
				: telemetryUnavailable
					? 'unavailable'
					: cleanSeries.length === 1
						? '1 sample'
						: 'sampling';
	$: compatibilityTone = tone;
</script>

<article class={`min-w-0 p-4 ${className}`.trim()} aria-label={`${label}: ${value}`} data-legacy-tone={compatibilityTone}>
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<div class="flex items-center gap-2">
				<span class={`h-1.5 w-1.5 rounded-full ${resourceClass.dot}`}></span>
				<p class="metric-label truncate">{label}</p>
			</div>
			<p class="metric-value mt-1 truncate text-xl font-semibold tracking-tight text-gray-950 dark:text-white">{value}</p>
		</div>
		{#if effectiveIndicator}
			<p class="metric-value shrink-0 text-xs font-semibold text-gray-500 dark:text-gray-400">{effectiveIndicator}</p>
		{/if}
	</div>

	<div class="relative mt-3 h-20 overflow-hidden rounded-md border border-gray-200 bg-white dark:border-gray-800 dark:bg-neutral-950">
		<svg class="h-full w-full" viewBox={`0 0 ${chartWidth} ${chartHeight}`} preserveAspectRatio="none" role="img" aria-hidden="true">
			<g class="stroke-gray-100/55 dark:stroke-neutral-800/45" stroke-width="0.8">
				<line x1="0" x2={chartWidth} y1={chartHeight * 0.25} y2={chartHeight * 0.25} />
				<line x1="0" x2={chartWidth} y1={chartHeight * 0.5} y2={chartHeight * 0.5} />
				<line x1="0" x2={chartWidth} y1={chartHeight * 0.75} y2={chartHeight * 0.75} />
				<line x1={chartWidth * 0.25} x2={chartWidth * 0.25} y1="0" y2={chartHeight} />
				<line x1={chartWidth * 0.5} x2={chartWidth * 0.5} y1="0" y2={chartHeight} />
				<line x1={chartWidth * 0.75} x2={chartWidth * 0.75} y1="0" y2={chartHeight} />
			</g>
			{#if areaPath}<path d={areaPath} class={resourceClass.fill} />{/if}
			{#if linePath}<path d={linePath} fill="none" class={resourceClass.stroke} stroke-width="1.25" vector-effect="non-scaling-stroke" />{/if}
			{#if currentPoint}<circle cx={currentPoint.x} cy={currentPoint.y} r="2" class={`${resourceClass.stroke} ${resourceClass.fill}`} stroke-width="1" />{/if}
		</svg>
		{#if chartMessage}
			<div class="pointer-events-none absolute inset-0 flex items-center justify-center px-4 text-center text-xs text-gray-400 dark:text-gray-600">{chartMessage}</div>
		{/if}
	</div>

	<div class="mt-2 flex items-center justify-between gap-3 text-xs text-gray-500 dark:text-gray-400">
		<p class="truncate">{detail}</p>
		<span class="shrink-0 font-mono">{effectiveRangeLabel}</span>
	</div>
</article>
