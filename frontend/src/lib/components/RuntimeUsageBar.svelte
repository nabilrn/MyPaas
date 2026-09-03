<script lang="ts">
	type Tone = 'cpu' | 'memory' | 'neutral';

	export let label = '';
	export let used = 0;
	export let limit = 0;
	export let valueLabel = '';
	export let tone: Tone = 'neutral';

	$: safeLimit = Math.max(limit, 0);
	$: ratio = safeLimit > 0 ? Math.max(0, Math.min(1, used / safeLimit)) : 0;
	$: percentage = Math.round(ratio * 100);
	$: trackClass = tone === 'cpu' ? 'runtime-track-cpu' : tone === 'memory' ? 'runtime-track-memory' : 'runtime-track-neutral';
	$: fillClass = tone === 'cpu' ? 'runtime-fill-cpu' : tone === 'memory' ? 'runtime-fill-memory' : 'runtime-fill-neutral';
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
		class={`mt-3 h-3 overflow-hidden border ${trackClass}`}
		role="progressbar"
		aria-label={label}
		aria-valuemin="0"
		aria-valuemax={safeLimit}
		aria-valuenow={Math.max(0, used)}
	>
		<div
			class={`h-full border-r ${fillClass}`}
			style={`width: ${ratio * 100}%`}
		></div>
	</div>
</article>

<style>
	.runtime-track-cpu {
		border-color: color-mix(in oklch, var(--chart-cpu) 28%, var(--app-border));
		background: var(--chart-cpu-soft);
	}

	.runtime-fill-cpu {
		border-color: color-mix(in oklch, var(--chart-cpu) 72%, transparent);
		background: var(--chart-cpu);
	}

	.runtime-track-memory {
		border-color: color-mix(in oklch, var(--chart-memory) 28%, var(--app-border));
		background: var(--chart-memory-soft);
	}

	.runtime-fill-memory {
		border-color: color-mix(in oklch, var(--chart-memory) 72%, transparent);
		background: var(--chart-memory);
	}

	.runtime-track-neutral {
		border-color: var(--app-border);
		background: transparent;
	}

	.runtime-fill-neutral {
		border-color: color-mix(in oklch, var(--app-border-strong) 70%, transparent);
		background: var(--app-border-strong);
	}
</style>
