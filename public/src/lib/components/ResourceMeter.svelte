<script lang="ts">
	export let label = '';
	export let value = '';
	export let detail = '';
	export let percent = 0;
	export let tone: 'neutral' | 'success' | 'info' | 'warning' | 'danger' = 'neutral';

	$: safePercent = Math.max(0, Math.min(100, percent));
	$: toneClass = {
		neutral: 'bg-gray-400 dark:bg-gray-500',
		success: 'bg-emerald-500',
		info: 'bg-sky-500',
		warning: 'bg-amber-500',
		danger: 'bg-red-500'
	}[tone];
	$: labelToneClass = {
		neutral: 'text-gray-500 dark:text-gray-400',
		success: 'text-emerald-700 dark:text-emerald-300',
		info: 'text-sky-700 dark:text-sky-300',
		warning: 'text-amber-700 dark:text-amber-300',
		danger: 'text-red-700 dark:text-red-300'
	}[tone];
</script>

<div>
	<div class="flex items-center justify-between gap-3">
		<p class="metric-label">{label}</p>
		<p class="font-mono text-xs font-medium {labelToneClass}">{safePercent.toFixed(0)}%</p>
	</div>
	<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800/80">
		<div class="h-full rounded-full {toneClass}" style={`width: ${safePercent}%`}></div>
	</div>
	<div class="mt-2 flex items-baseline justify-between gap-3">
		<p class="truncate text-sm font-semibold text-gray-950 dark:text-white">{value}</p>
		{#if detail}
			<p class="truncate text-xs text-gray-500 dark:text-gray-400">{detail}</p>
		{/if}
	</div>
</div>
