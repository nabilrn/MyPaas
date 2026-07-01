<script lang="ts">
	import type { ProjectStatus, DeployStatus } from '$types';

	export let status: ProjectStatus | DeployStatus;
	export let pulse = false;

	const cfg: Record<string, { label: string; classes: string }> = {
		running:     { label: 'Running',     classes: 'bg-green-100  text-green-700  dark:bg-green-900/30  dark:text-green-400'  },
		stopped:     { label: 'Stopped',     classes: 'bg-gray-100   text-gray-600   dark:bg-gray-800      dark:text-gray-400'   },
		crashed:     { label: 'Crashed',     classes: 'bg-red-100    text-red-700    dark:bg-red-900/30    dark:text-red-400'    },
		building:    { label: 'Building',    classes: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' },
		pending:     { label: 'Pending',     classes: 'bg-gray-100   text-gray-600   dark:bg-gray-800      dark:text-gray-400'   },
		queued:      { label: 'Queued',      classes: 'bg-blue-100   text-blue-700   dark:bg-blue-900/30   dark:text-blue-400'   },
		cloning:     { label: 'Cloning',     classes: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' },
		starting:    { label: 'Starting',    classes: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' },
		failed:      { label: 'Failed',      classes: 'bg-red-100    text-red-700    dark:bg-red-900/30    dark:text-red-400'    },
		rolled_back: { label: 'Rolled back', classes: 'bg-gray-100   text-gray-600   dark:bg-gray-800      dark:text-gray-400'   }
	};

	$: c = cfg[status] ?? { label: status, classes: 'bg-gray-100 text-gray-600' };
	$: isPulsing = pulse && ['building', 'cloning', 'starting', 'queued'].includes(status);
</script>

<span class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium {c.classes}">
	{#if isPulsing}
		<span class="relative flex h-2 w-2">
			<span class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-75 {c.classes}"></span>
			<span class="relative inline-flex h-2 w-2 rounded-full bg-current"></span>
		</span>
	{:else}
		<span class="h-1.5 w-1.5 rounded-full bg-current"></span>
	{/if}
	{c.label}
</span>
