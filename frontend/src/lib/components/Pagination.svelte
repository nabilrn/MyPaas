<script lang="ts">
	import ActionButton from './ActionButton.svelte';

	export let page = 0;
	export let pageSize = 20;
	export let totalShown = 0;
	export let hasNext = false;
	export let loading = false;
	export let label = 'Rows';

	const dispatchPrev = () => {
		if (page > 0 && !loading) {
			page -= 1;
		}
	};
	const dispatchNext = () => {
		if (hasNext && !loading) {
			page += 1;
		}
	};

	$: start = totalShown === 0 ? 0 : page * pageSize + 1;
	$: end = page * pageSize + totalShown;
</script>

<div
	class="flex flex-col gap-3 border-t border-gray-100 bg-gray-50/70 px-5 py-3 text-sm dark:border-gray-800 dark:bg-gray-900/70 sm:flex-row sm:items-center sm:justify-between"
	role="navigation"
	aria-label={`${label} pagination`}
>
	<p class="text-xs text-gray-500 dark:text-gray-400" aria-live="polite">
		{label}: {start}-{end}{hasNext ? '+' : ''}
	</p>
	<div class="flex items-center gap-2">
		<ActionButton
			on:click={dispatchPrev}
			disabled={page === 0 || loading}
			ariaLabel={`Previous ${label.toLowerCase()} page`}
			variant="secondary"
			size="xs"
		>
			Previous
		</ActionButton>
		<span class="min-w-16 text-center text-xs font-medium text-gray-500 dark:text-gray-400" aria-live="polite">Page {page + 1}</span>
		<ActionButton
			on:click={dispatchNext}
			disabled={!hasNext || loading}
			ariaLabel={`Next ${label.toLowerCase()} page`}
			variant="secondary"
			size="xs"
		>
			Next
		</ActionButton>
	</div>
</div>
