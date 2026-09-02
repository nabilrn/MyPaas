<script lang="ts">
	import type { ProjectOperationalTone } from '$lib/utils/project-operational-state';

	export let status = '';
	export let label: string | undefined = undefined;
	export let tone: ProjectOperationalTone | undefined = undefined;

	$: normalized = status.toLowerCase();
	$: defaultLabel = normalized === 'running'
		? 'Running'
		: normalized === 'building'
			? 'Building'
			: normalized === 'crashed'
				? 'Crashed'
				: normalized === 'stopped'
					? 'Stopped'
					: normalized === 'pending'
						? 'Pending'
						: status || 'Unknown';
	$: resolvedTone = tone ?? (normalized === 'running'
		? 'success'
		: normalized === 'building'
			? 'warning'
			: normalized === 'crashed'
				? 'danger'
				: normalized === 'pending'
					? 'info'
					: 'neutral');
	$: dotClass = resolvedTone === 'success'
		? 'bg-emerald-500'
		: resolvedTone === 'warning'
			? 'bg-amber-500'
			: resolvedTone === 'danger'
				? 'bg-red-500'
				: resolvedTone === 'info'
					? 'bg-sky-500'
					: 'bg-gray-400 dark:bg-gray-500';
</script>

<span class="inline-flex items-center gap-2 whitespace-nowrap text-sm text-gray-700 dark:text-gray-300">
	<span class={`status-dot ${dotClass}`} aria-hidden="true"></span>
	{label ?? defaultLabel}
</span>
