<script lang="ts">
	import { HardDrive } from '@lucide/svelte';

	export let label = 'Storage';
	export let value = 'Unavailable';
	export let detail = 'Host telemetry unavailable';
	export let percent = 0;
	export let className = '';

	$: usedPercent = Math.min(Math.max(Number.isFinite(percent) ? percent : 0, 0), 100);
	$: available = !/unavailable/i.test(`${value} ${detail}`);
	$: fillClass = usedPercent >= 90
		? 'bg-red-500 dark:bg-red-400'
		: usedPercent >= 80
			? 'bg-orange-500 dark:bg-orange-400'
			: 'bg-amber-400 dark:bg-amber-300';
</script>

<article
	class={`storage-capacity-metric min-w-0 p-4 ${className}`.trim()}
	aria-label={`${label}: ${value}`}
	data-storage-capacity
>
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<div class="flex items-center gap-2">
				<HardDrive class="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400" aria-hidden="true" />
				<p class="metric-label truncate">{label}</p>
			</div>
			<p class="metric-value mt-1 truncate text-lg font-semibold tracking-tight text-gray-950 dark:text-white">{available ? value : 'Unavailable'}</p>
		</div>
		{#if available}
			<p class="metric-value shrink-0 text-xs font-medium text-gray-500 dark:text-gray-400">{usedPercent.toFixed(0)}%</p>
		{/if}
	</div>

	<div class="flex h-32 items-center">
		<div class="w-full">
			<div
				class="h-3 overflow-hidden rounded-sm border border-gray-300 bg-gray-100 dark:border-neutral-700 dark:bg-neutral-800"
				role={available ? 'progressbar' : undefined}
				aria-label={available ? `${label} used` : undefined}
				aria-valuemin={available ? 0 : undefined}
				aria-valuemax={available ? 100 : undefined}
				aria-valuenow={available ? Math.round(usedPercent) : undefined}
			>
				{#if available}
					<div class={`h-full transition-[width] duration-300 motion-reduce:transition-none ${fillClass}`} style={`width: ${usedPercent}%`}></div>
				{/if}
			</div>
			<p class="mt-2 truncate text-xs text-gray-500 dark:text-gray-400">{detail}</p>
		</div>
	</div>
</article>

<style>
	/* The host resources surface contains exactly the four capacity metrics.
	   Keep Storage in that same row on desktop while preserving the existing
	   two-column tablet layout. */
	@media (min-width: 1280px) {
		:global(.grid:has(> .storage-capacity-metric)) {
			grid-template-columns: repeat(4, minmax(0, 1fr));
		}

		:global(.grid:has(> .storage-capacity-metric) > .storage-capacity-metric) {
			grid-column: auto !important;
		}
	}
</style>
