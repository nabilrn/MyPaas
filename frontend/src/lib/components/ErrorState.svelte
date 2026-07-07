<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import ActionButton from './ActionButton.svelte';

	export let title = 'Could not load data';
	export let message = 'Something went wrong.';
	export let retryLabel = 'Retry';

	const dispatch = createEventDispatcher<{ retry: void }>();
</script>

<div class="p-5">
	<div class="rounded-md border border-red-200 bg-red-50/80 p-4 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/25 dark:text-red-200">
		<p class="font-semibold">{title}</p>
		<p class="mt-1 max-w-2xl text-red-600 dark:text-red-200/80">{message}</p>
		<div class="mt-3">
			<ActionButton variant="secondary" size="xs" on:click={() => dispatch('retry')}>
				{retryLabel}
			</ActionButton>
		</div>
	</div>
</div>
