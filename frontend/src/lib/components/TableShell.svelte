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
	// Retained as a compatibility prop for existing callers. TableShell no longer
	// owns initial loading visuals; authenticated main content owns that state.
	export let loadingRows = 3;
	export let contentClass = 'overflow-x-auto';

	const dispatch = createEventDispatcher<{ retry: void }>();
</script>

<section class="surface min-w-0 overflow-hidden" aria-busy={loading}>
	{#if title || description || $$slots.actions}
		<div class="panel-header flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
			<div class="min-w-0">
				{#if title}
					<h2 class="panel-title">{title}</h2>
				{/if}
				{#if description}
					<p class="panel-description">{description}</p>
				{/if}
			</div>
			<div class="flex shrink-0 flex-wrap items-center gap-2">
				<slot name="actions" />
			</div>
		</div>
	{/if}

	{#if error && !loading}
		<ErrorState message={error} on:retry={() => dispatch('retry')} />
	{:else if empty && !loading}
		<EmptyState title={emptyTitle} description={emptyDescription} compact />
	{:else}
		<slot name="notice" />
		<div class={contentClass}>
			<slot />
		</div>
		<slot name="footer" />
	{/if}
</section>
