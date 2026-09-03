<script lang="ts">
	export let label = '';
	export let used = 0;
	export let limit = 0;
	export let valueLabel = '';
	export let allocationLabel = '';

	$: safeLimit = Math.max(limit, 0);
	$: ratio = safeLimit > 0 ? Math.max(0, Math.min(1, used / safeLimit)) : 0;
	$: percentage = Math.round(ratio * 100);
</script>

<article class="min-w-0 bg-white px-4 py-3 dark:bg-neutral-950" aria-label={label}>
	<div class="flex items-start justify-between gap-4">
		<div class="min-w-0">
			<p class="text-[13px] font-medium text-gray-700 dark:text-gray-300">{label}</p>
			<p class="mt-1 truncate text-sm font-semibold text-gray-950 dark:text-white">{valueLabel}</p>
		</div>
		<span class="shrink-0 font-mono text-[11px] text-gray-400 dark:text-gray-500">{percentage}%</span>
	</div>

	<div
		class="mt-3 h-3 overflow-hidden border border-gray-200 bg-transparent dark:border-neutral-800"
		role="progressbar"
		aria-label={label}
		aria-valuemin="0"
		aria-valuemax={safeLimit}
		aria-valuenow={Math.max(0, used)}
	>
		<div
			class="h-full border-r border-gray-400/40 bg-gray-300/70 dark:border-neutral-500/40 dark:bg-neutral-700/80"
			style={`width: ${ratio * 100}%`}
		></div>
	</div>

	<div class="mt-1.5 flex items-center justify-between gap-3 text-[11px] text-gray-400 dark:text-gray-500">
		<span>Allocated resource</span>
		<span class="font-mono">{allocationLabel}</span>
	</div>
</article>
