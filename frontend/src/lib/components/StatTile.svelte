<script lang="ts">
	export let label = '';
	export let value = '';
	export let detail = '';
	export let tone: 'neutral' | 'success' | 'info' | 'warning' | 'danger' = 'neutral';

	$: toneClass = {
		neutral: 'bg-gray-400',
		success: 'bg-emerald-500',
		info: 'bg-sky-500',
		warning: 'bg-amber-500',
		danger: 'bg-red-500'
	}[tone];
	$: valueClass = {
		neutral: 'text-gray-950 dark:text-white',
		success: 'text-emerald-700 dark:text-emerald-200',
		info: 'text-sky-700 dark:text-sky-200',
		warning: 'text-amber-700 dark:text-amber-200',
		danger: 'text-red-700 dark:text-red-200'
	}[tone];
</script>

<div class="soft-panel min-w-0 p-4">
	<div class="flex items-center gap-2">
		<span class={`h-1.5 w-1.5 rounded-full ${toneClass}`}></span>
		<p class="metric-label truncate">{label}</p>
	</div>
	<p class={`mt-2 truncate text-2xl font-semibold tracking-tight ${valueClass}`}>{value}</p>
	{#if detail}
		<p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{detail}</p>
	{/if}
	{#if $$slots.default}
		<div class="mt-3">
			<slot />
		</div>
	{/if}
</div>
