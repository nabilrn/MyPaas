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
			: 'bg-amber-500 dark:bg-amber-400';
</script>

<article class={`min-w-0 p-4 sm:p-5 ${className}`.trim()} aria-label={`${label}: ${value}`}>
	<div class="flex items-start gap-3 sm:gap-4">
		<div class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center text-gray-500 dark:text-gray-400">
			<HardDrive class="h-6 w-6" aria-hidden="true" />
		</div>
		<div class="min-w-0 flex-1">
			<p class="metric-label">{label}</p>
			{#if available}
				<p class="metric-value mt-1 text-xl font-semibold tracking-tight text-gray-950 dark:text-white sm:text-2xl">{value}</p>
				<p class="metric-value mt-3 text-xl font-semibold tracking-tight text-gray-950 dark:text-white">{usedPercent.toFixed(0)}%</p>
				<div
					class="mt-3 h-2 overflow-hidden rounded-full bg-gray-100 ring-1 ring-inset ring-gray-200 dark:bg-neutral-800 dark:ring-neutral-700"
					role="progressbar"
					aria-label={`${label} used`}
					aria-valuemin="0"
					aria-valuemax="100"
					aria-valuenow={Math.round(usedPercent)}
				>
					<div class={`h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none ${fillClass}`} style={`width: ${usedPercent}%`}></div>
				</div>
				<p class="mt-3 text-[13px] text-gray-500 dark:text-gray-400">{detail}</p>
			{:else}
				<p class="metric-value mt-1 text-xl font-semibold tracking-tight text-gray-950 dark:text-white">Unavailable</p>
				<div class="mt-3 h-2 rounded-full bg-gray-100 ring-1 ring-inset ring-gray-200 dark:bg-neutral-800 dark:ring-neutral-700"></div>
				<p class="mt-3 text-[13px] text-gray-500 dark:text-gray-400">{detail}</p>
			{/if}
		</div>
	</div>
</article>
