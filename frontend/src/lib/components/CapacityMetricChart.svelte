<script lang="ts">
	import StorageCapacityMetric from './StorageCapacityMetric.svelte';
	import { deriveAdaptiveMetricDomain } from '$lib/utils/host-telemetry';

	type Resource = 'memory' | 'cpu' | 'storage' | 'network' | 'neutral';
	type ChartPoint = { x: number; y: number; encoded: string };

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

	const chartWidth = 240;
	const chartHeight = 82;
	const curveTension = 0.72;
	let inferredResource: Resource = 'neutral';
	const resourceClasses = {
		neutral: {
			dot: 'bg-gray-400/75 dark:bg-gray-400/70',
			stroke: 'stroke-gray-500/60 dark:stroke-gray-300/55',
			fill: 'fill-gray-400/[0.07] dark:fill-gray-300/[0.08]',
			point: 'fill-gray-500/75 dark:fill-gray-300/70'
		},
		memory: {
			dot: 'bg-emerald-400/70 dark:bg-emerald-300/65',
			stroke: 'stroke-emerald-500/60 dark:stroke-emerald-300/55',
			fill: 'fill-emerald-400/[0.07] dark:fill-emerald-300/[0.08]',
			point: 'fill-emerald-500/75 dark:fill-emerald-300/70'
		},
		cpu: {
			dot: 'bg-sky-400/70 dark:bg-sky-300/65',
			stroke: 'stroke-sky-500/60 dark:stroke-sky-300/55',
			fill: 'fill-sky-400/[0.07] dark:fill-sky-300/[0.08]',
			point: 'fill-sky-500/75 dark:fill-sky-300/70'
		},
		storage: {
			dot: 'bg-amber-400/70 dark:bg-amber-300/65',
			stroke: 'stroke-amber-500/58 dark:stroke-amber-300/52',
			fill: 'fill-amber-400/[0.06] dark:fill-amber-300/[0.07]',
			point: 'fill-amber-500/70 dark:fill-amber-300/65'
		},
		network: {
			dot: 'bg-violet-400/70 dark:bg-violet-300/65',
			stroke: 'stroke-violet-500/60 dark:stroke-violet-300/55',
			fill: 'fill-violet-400/[0.07] dark:fill-violet-300/[0.08]',
			point: 'fill-violet-500/75 dark:fill-violet-300/70'
		}
	} as const;

	function formatDomainValue(value: number, span: number) {
		return span < 10 ? value.toFixed(1) : Math.round(value).toString();
	}

	function clamp(value: number, min: number, max: number) {
		return Math.max(min, Math.min(max, value));
	}

	function buildSmoothPath(input: ChartPoint[]) {
		if (input.length < 2) return '';
		let path = `M ${input[0].encoded}`;

		for (let index = 0; index < input.length - 1; index += 1) {
			const p0 = input[index - 1] ?? input[index];
			const p1 = input[index];
			const p2 = input[index + 1];
			const p3 = input[index + 2] ?? p2;

			const cp1x = clamp(p1.x + ((p2.x - p0.x) / 6) * curveTension, 0, chartWidth);
			const cp1y = clamp(p1.y + ((p2.y - p0.y) / 6) * curveTension, 0, chartHeight);
			const cp2x = clamp(p2.x - ((p3.x - p1.x) / 6) * curveTension, 0, chartWidth);
			const cp2y = clamp(p2.y - ((p3.y - p1.y) / 6) * curveTension, 0, chartHeight);

			path += ` C ${cp1x.toFixed(2)},${cp1y.toFixed(2)} ${cp2x.toFixed(2)},${cp2y.toFixed(2)} ${p2.encoded}`;
		}

		return path;
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
	$: requestedIndicator = indicator || (percent !== null && Number.isFinite(percent) ? `${Math.max(0, percent).toFixed(0)}%` : '');
	$: effectiveIndicator = /%$/.test(value.trim()) ? value.trim() : requestedIndicator;
	$: rollingHistory = series.length >= 2 && cleanSeries.length >= 2;
	$: domain = deriveAdaptiveMetricDomain(cleanSeries, maxValue);
	$: domainMin = domain.min;
	$: domainMax = domain.max;
	$: domainSpan = Math.max(0.0001, domainMax - domainMin);
	$: points = cleanSeries.map((sample, index) => {
		const x = cleanSeries.length === 1 ? chartWidth - 4 : (index / Math.max(1, cleanSeries.length - 1)) * chartWidth;
		const level = Math.max(0, Math.min(1, (sample - domainMin) / domainSpan));
		const y = 4 + (1 - level) * (chartHeight - 8);
		return { x, y, encoded: `${x.toFixed(2)},${y.toFixed(2)}` };
	});
	$: linePath = buildSmoothPath(points);
	$: areaPath = points.length >= 2 && linePath
		? `${linePath} L ${points[points.length - 1].x.toFixed(2)},${chartHeight} L ${points[0].x.toFixed(2)},${chartHeight} Z`
		: '';
	$: resourceClass = resourceClasses[inferredResource] ?? resourceClasses.neutral;
	$: currentPoint = points.length === 1 ? points[0] : null;
	$: latestPoint = rollingHistory ? points[points.length - 1] : null;
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
	$: storagePercent = cleanSeries.length > 0
		? cleanSeries[cleanSeries.length - 1]
		: percent !== null && Number.isFinite(percent)
			? percent
			: Number.parseFloat(effectiveIndicator) || 0;
</script>

{#if inferredResource === 'storage'}
	<StorageCapacityMetric {label} {value} {detail} percent={storagePercent} {className} />
{:else}
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

		<div class="relative mt-3 h-[5.75rem] overflow-hidden rounded-md border border-gray-200/90 bg-white dark:border-gray-800/90 dark:bg-neutral-950">
			<svg class="h-full w-full" viewBox={`0 0 ${chartWidth} ${chartHeight}`} preserveAspectRatio="none" role="img" aria-hidden="true">
				<g class="stroke-gray-100/45 dark:stroke-neutral-800/35" stroke-width="0.7">
					<line x1="0" x2={chartWidth} y1={chartHeight * 0.25} y2={chartHeight * 0.25} />
					<line x1="0" x2={chartWidth} y1={chartHeight * 0.5} y2={chartHeight * 0.5} />
					<line x1="0" x2={chartWidth} y1={chartHeight * 0.75} y2={chartHeight * 0.75} />
				</g>
				{#if areaPath}<path d={areaPath} class={resourceClass.fill} />{/if}
				{#if linePath}
					<path
						d={linePath}
						fill="none"
						class={resourceClass.stroke}
						stroke-width="1.45"
						stroke-linecap="round"
						stroke-linejoin="round"
						vector-effect="non-scaling-stroke"
					/>
				{/if}
				{#if latestPoint}
					<circle cx={latestPoint.x} cy={latestPoint.y} r="1.45" class={resourceClass.point} />
				{:else if currentPoint}
					<circle cx={currentPoint.x} cy={currentPoint.y} r="1.65" class={resourceClass.point} />
				{/if}
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
{/if}
