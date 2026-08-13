<script lang="ts">
	export let status = '';

	$: normalized = status.toLowerCase();
	$: label = normalized === 'running'
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
	$: dotClass = normalized === 'running'
		? 'bg-emerald-500'
		: normalized === 'building'
			? 'bg-amber-500'
			: normalized === 'crashed'
				? 'bg-red-500'
				: normalized === 'pending'
					? 'bg-sky-500'
					: 'bg-gray-400 dark:bg-gray-500';
</script>

<span class="inline-flex items-center gap-2 whitespace-nowrap text-sm text-gray-700 dark:text-gray-300">
	<span class={`status-dot ${dotClass}`} aria-hidden="true"></span>
	{label}
</span>
