<script lang="ts">
	export let label = 'Storage';
	export let value = 'Unavailable';
	export let detail = 'Host telemetry unavailable';
	export let percent = 0;
	export let className = '';

	$: usedPercent = Math.min(Math.max(Number.isFinite(percent) ? percent : 0, 0), 100);
	$: available = !/unavailable/i.test(`${value} ${detail}`);
	$: fillTop = 76 - (56 * usedPercent) / 100;
	$: fillClass = usedPercent >= 90
		? 'fill-red-500/70 dark:fill-red-400/60'
		: usedPercent >= 80
			? 'fill-orange-600/70 dark:fill-orange-300/60'
			: 'fill-orange-500/65 dark:fill-orange-400/55';
	$: fillStrokeClass = usedPercent >= 90
		? 'stroke-red-600/55 dark:stroke-red-300/50'
		: 'stroke-orange-600/55 dark:stroke-orange-300/50';
</script>

<article class={`grid h-full min-h-40 min-w-0 grid-cols-[minmax(0,1fr)_5.5rem] items-center gap-4 p-4 ${className}`.trim()} aria-label={`${label}: ${value}`}>
	<div class="min-w-0">
		<p class="metric-label truncate">{label}</p>
		<p class="metric-value mt-2 truncate text-xl font-semibold tracking-tight text-gray-950 dark:text-white">{value}</p>
		{#if available}
			<p class="metric-value mt-4 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{usedPercent.toFixed(0)}%</p>
		{:else}
			<p class="mt-4 text-sm font-medium text-gray-400 dark:text-gray-500">No sample</p>
		{/if}
		<p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">{detail}</p>
	</div>

	<div
		class="flex items-center justify-center"
		role={available ? 'progressbar' : undefined}
		aria-label={available ? `${label} used` : undefined}
		aria-valuemin={available ? 0 : undefined}
		aria-valuemax={available ? 100 : undefined}
		aria-valuenow={available ? Math.round(usedPercent) : undefined}
	>
		<svg viewBox="0 0 96 104" class="h-24 w-20" aria-hidden="true">
			<ellipse cx="48" cy="76" rx="28" ry="7" fill="none" class="stroke-gray-300/70 dark:stroke-gray-700/70" stroke-width="1" />
			{#if available && usedPercent > 0}
				<rect x="20.75" y={fillTop} width="54.5" height={76 - fillTop} class={fillClass} />
				<ellipse cx="48" cy={fillTop} rx="27.25" ry="6.6" class={`${fillClass} ${fillStrokeClass}`} stroke-width="0.8" />
				<ellipse cx="48" cy="76" rx="27.25" ry="6.6" class={fillClass} />
				<path d={`M24 ${fillTop} C 34 ${fillTop - 3.2} 62 ${fillTop - 3.2} 72 ${fillTop}`} fill="none" class="stroke-white/40 dark:stroke-white/25" stroke-width="0.8" stroke-linecap="round" />
			{/if}
			<ellipse cx="48" cy="20" rx="28" ry="8" fill="none" class="stroke-gray-500 dark:stroke-gray-400" stroke-width="1.4" />
			<path d="M20 20v56c0 4.4 12.5 8 28 8s28-3.6 28-8V20" fill="none" class="stroke-gray-500 dark:stroke-gray-400" stroke-width="1.4" />
			<path d="M20 76c0 4.4 12.5 8 28 8s28-3.6 28-8" fill="none" class="stroke-gray-500 dark:stroke-gray-400" stroke-width="1.4" />
			{#if !available}
				<path d="M32 49h32" class="stroke-gray-300 dark:stroke-gray-700" stroke-width="1.5" stroke-dasharray="3 3" />
			{/if}
		</svg>
	</div>
</article>
