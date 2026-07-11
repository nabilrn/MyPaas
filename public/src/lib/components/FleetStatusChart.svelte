<script lang="ts">
	type FleetSegment = {
		label: string;
		value: number;
		tone: 'success' | 'info' | 'warning' | 'danger' | 'neutral';
	};

	export let segments: FleetSegment[] = [];
	export let title = 'Fleet health';
	export let subtitle = '';

	const radius = 38;
	const circumference = 2 * Math.PI * radius;

	$: total = segments.reduce((sum, item) => sum + item.value, 0);
	$: visibleSegments = segments.filter((item) => item.value > 0);
	$: activeLabel = total > 0 ? `${total} total` : 'No projects';
	$: ringSegments = visibleSegments.map((segment, index) => {
		const previous = visibleSegments.slice(0, index).reduce((sum, item) => sum + item.value, 0);
		const length = total > 0 ? (segment.value / total) * circumference : 0;
		const offset = total > 0 ? (previous / total) * circumference : 0;
		return { ...segment, length, offset };
	});
	$: toneClass = {
		success: 'stroke-emerald-500',
		info: 'stroke-sky-500',
		warning: 'stroke-amber-500',
		danger: 'stroke-red-500',
		neutral: 'stroke-gray-400'
	};
	$: dotClass = {
		success: 'bg-emerald-500',
		info: 'bg-sky-500',
		warning: 'bg-amber-500',
		danger: 'bg-red-500',
		neutral: 'bg-gray-400'
	};
</script>

<section class="surface h-full overflow-hidden p-5">
	<div class="flex items-start justify-between gap-4">
		<div>
			<h2 class="text-sm font-semibold text-gray-950 dark:text-white">{title}</h2>
			{#if subtitle}
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{subtitle}</p>
			{/if}
		</div>
		<span class="shrink-0 text-xs font-medium text-gray-500 dark:text-gray-400">{activeLabel}</span>
	</div>

	<div class="mt-5 grid gap-5 sm:grid-cols-[9rem_minmax(0,1fr)] sm:items-center">
		<div class="relative mx-auto h-32 w-32">
			<svg viewBox="0 0 100 100" class="h-full w-full -rotate-90" role="img" aria-label={title}>
				<circle cx="50" cy="50" r={radius} fill="none" class="stroke-gray-100 dark:stroke-gray-800" stroke-width="10" />
				{#each ringSegments as segment}
					<circle
						cx="50"
						cy="50"
						r={radius}
						fill="none"
						class={toneClass[segment.tone]}
						stroke-width="10"
						stroke-linecap="round"
						stroke-dasharray={`${segment.length} ${circumference - segment.length}`}
						stroke-dashoffset={-segment.offset}
					/>
				{/each}
			</svg>
			<div class="absolute inset-0 flex flex-col items-center justify-center text-center">
				<span class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{total}</span>
				<span class="text-[11px] text-gray-500 dark:text-gray-400">projects</span>
			</div>
		</div>

		<div class="space-y-3">
			{#each segments as segment}
				<div>
					<div class="flex items-center justify-between gap-3 text-xs">
						<span class="inline-flex min-w-0 items-center gap-2 text-gray-600 dark:text-gray-300">
							<span class="h-2 w-2 shrink-0 rounded-full {dotClass[segment.tone]}"></span>
							<span class="truncate">{segment.label}</span>
						</span>
						<span class="font-mono font-medium text-gray-950 dark:text-white">{segment.value}</span>
					</div>
					<div class="mt-1 h-1 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
						<div
							class="h-full rounded-full {dotClass[segment.tone]}"
							style={`width: ${total > 0 ? Math.max(2, (segment.value / total) * 100) : 0}%`}
						></div>
					</div>
				</div>
			{/each}
		</div>
	</div>
</section>
