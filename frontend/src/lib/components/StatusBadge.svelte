<script lang="ts">
	import type { ProjectStatus, DeployStatus } from '$types';

	export let status: ProjectStatus | DeployStatus;
	export let pulse = false;

	const cfg: Record<string, { label: string; classes: string; dot: string }> = {
		running:     { label: 'Running',     classes: 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-300', dot: 'bg-emerald-500' },
		stopped:     { label: 'Stopped',     classes: 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400', dot: 'bg-gray-400' },
		crashed:     { label: 'Crashed',     classes: 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300', dot: 'bg-red-500' },
		building:    { label: 'Building',    classes: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300', dot: 'bg-amber-500' },
		pending:     { label: 'Pending',     classes: 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400', dot: 'bg-gray-400' },
		queued:      { label: 'Queued',      classes: 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-300', dot: 'bg-sky-500' },
		cloning:     { label: 'Cloning',     classes: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300', dot: 'bg-amber-500' },
		starting:    { label: 'Starting',    classes: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300', dot: 'bg-amber-500' },
		failed:      { label: 'Failed',      classes: 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300', dot: 'bg-red-500' },
		rolled_back: { label: 'Rolled back', classes: 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400', dot: 'bg-gray-400' }
	};

	$: c = cfg[status] ?? { label: status, classes: 'border-gray-200 bg-gray-50 text-gray-600', dot: 'bg-gray-400' };
	$: isPulsing = pulse && ['building', 'cloning', 'starting', 'queued'].includes(status);
</script>

<span class="inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-[11px] font-medium leading-none {c.classes}">
	{#if isPulsing}
		<span class="relative flex h-1.5 w-1.5">
			<span class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-60 {c.dot}"></span>
			<span class="relative inline-flex h-1.5 w-1.5 rounded-full {c.dot}"></span>
		</span>
	{:else}
		<span class="h-1.5 w-1.5 rounded-full {c.dot}"></span>
	{/if}
	{c.label}
</span>
