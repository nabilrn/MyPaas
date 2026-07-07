<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import EmptyState from './EmptyState.svelte';
	import ErrorState from './ErrorState.svelte';

	export let title = '';
	export let description = '';
	export let loading = false;
	export let error = '';
	export let empty = false;
	export let emptyTitle = 'No rows yet.';
	export let emptyDescription = '';
	export let loadingRows = 3;
	export let contentClass = 'overflow-x-auto';

	const dispatch = createEventDispatcher<{ retry: void }>();
	const skeletonRows = Array.from({ length: loadingRows }, (_, index) => index);
</script>

<section class="surface min-w-0 overflow-hidden">
	{#if title || description || $$slots.actions}
		<div class="panel-header flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<div class="min-w-0">
				{#if title}
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">{title}</h2>
				{/if}
				{#if description}
					<p class="mt-0.5 max-w-2xl text-xs text-gray-500 dark:text-gray-400">{description}</p>
				{/if}
			</div>
			<div class="flex shrink-0 flex-wrap items-center gap-2">
				<slot name="actions" />
			</div>
		</div>
	{/if}

	{#if loading}
		<div class="space-y-3 p-5" aria-busy="true" aria-live="polite">
			{#each skeletonRows as _}
				<div class="h-12 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800/80"></div>
			{/each}
		</div>
	{:else if error}
		<ErrorState message={error} on:retry={() => dispatch('retry')} />
	{:else if empty}
		<EmptyState title={emptyTitle} description={emptyDescription} compact />
	{:else}
		<slot name="notice" />
		<div class={contentClass}>
			<slot />
		</div>
		<slot name="footer" />
	{/if}
</section>
