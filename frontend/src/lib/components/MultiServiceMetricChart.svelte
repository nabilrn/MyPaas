<script lang="ts">
	type MetricSeries = {
		service: string;
		values: number[];
		value: string;
	};

	export let label = '';
	export let series: MetricSeries[] = [];
	export let suffix = '';
	export let maxValue: number | null = null;
	export let heightClass = 'h-56';

	const width = 720;
	const height = 180;
	const palette = [
		'stroke-sky-500 dark:stroke-sky-300',
		'stroke-emerald-500 dark:stroke-emerald-300',
		'stroke-violet-500 dark:stroke-violet-300',
		'stroke-amber-500 dark:stroke-amber-300',
		'stroke-rose-500 dark:stroke-rose-300',
		'stroke-cyan-500 dark:stroke-cyan-300',
		'stroke-fuchsia-500 dark:stroke-fuchsia-300',
		'stroke-lime-600 dark:stroke-lime-300'
	];
	const dotPalette = [
		'bg-sky-500 dark:bg-sky-300',
		'bg-emerald-500 dark:bg-emerald-300',
		'bg-violet-500 dark:bg-violet-300',
		'bg-amber-500 dark:bg-amber-300',
		'bg-rose-500 dark:bg-rose-300',
		'bg-cyan-500 dark:bg-cyan-300',
		'bg-fuchsia-500 dark:bg-fuchsia-300',
		'bg-lime-600 dark:bg-lime-300'
	];

	$: allValues = series.flatMap((item) => item.values).filter((value) => Number.isFinite(value));
	$: observedMax = allValues.length > 0 ? Math.max(...allValues) : 0;
	$: domainMax = maxValue !== null
		? maxValue
		: Math.max(1, Math.ceil((observedMax * 1.15) / 5) * 5);
	$: paths = series.map((item, index) => ({
		...item,
		path: buildPath(item.values, domainMax),
		strokeClass: palette[index % palette.length],
		dotClass: dotPalette[index % dotPalette.length]
	}));

	function buildPath(values: number[], ceiling: number) {
		const clean = values.filter((value) => Number.isFinite(value));
		if (clean.length === 0) return '';
		return clean.map((value, index) => {
			const x = clean.length === 1 ? width : (index / Math.max(1, clean.length - 1)) * width;
			const ratio = Math.max(0, Math.min(1, value / Math.max(ceiling, 0.0001)));
			const y = 8 + (1 - ratio) * (height - 16);
			return `${index === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`;
		}).join(' ');
	}
</script>

<article class="min-w-0 bg-white p-5 dark:bg-neutral-900" aria-label={label}>
	<div class="flex flex-wrap items-start justify-between gap-x-5 gap-y-2">
		<div>
			<p class="text-sm font-medium text-gray-700 dark:text-gray-300">{label}</p>
			<p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">{series.length} service{series.length === 1 ? '' : 's'}</p>
		</div>
		<div class="flex max-w-full flex-wrap justify-end gap-x-4 gap-y-1.5">
			{#each paths as item}
				<div class="inline-flex min-w-0 items-center gap-1.5 text-xs">
					<span class={`h-2 w-2 shrink-0 rounded-full ${item.dotClass}`}></span>
					<span class="max-w-32 truncate text-gray-500 dark:text-gray-400" title={item.service}>{item.service}</span>
					<span class="metric-value font-medium text-gray-950 dark:text-white">{item.value}</span>
				</div>
			{/each}
		</div>
	</div>

	<div class={`relative mt-4 ${heightClass} overflow-hidden rounded-md border border-gray-200/70 bg-white dark:border-neutral-800/80 dark:bg-neutral-950`}>
		<svg class="h-full w-full" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" role="img" aria-hidden="true">
			<g class="stroke-gray-100/70 dark:stroke-neutral-800/55" stroke-width="0.8">
				<line x1="0" x2={width} y1={height * 0.25} y2={height * 0.25} />
				<line x1="0" x2={width} y1={height * 0.5} y2={height * 0.5} />
				<line x1="0" x2={width} y1={height * 0.75} y2={height * 0.75} />
			</g>
			{#each paths as item}
				{#if item.path}
					<path
						d={item.path}
						fill="none"
						class={item.strokeClass}
						stroke-width="1.8"
						stroke-linecap="round"
						stroke-linejoin="round"
						vector-effect="non-scaling-stroke"
					/>
				{/if}
			{/each}
		</svg>
		{#if paths.every((item) => !item.path)}
			<div class="absolute inset-0 flex items-center justify-center text-sm text-gray-400 dark:text-gray-600">Waiting for samples</div>
		{/if}
	</div>

	<div class="mt-2 flex justify-between gap-3 text-xs text-gray-400 dark:text-gray-500">
		<span>rolling samples</span>
		<span class="font-mono">0–{domainMax.toFixed(domainMax < 10 ? 1 : 0)}{suffix}</span>
	</div>
</article>
