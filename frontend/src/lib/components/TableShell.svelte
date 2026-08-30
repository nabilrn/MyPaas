<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import EmptyState from './EmptyState.svelte';
	import ErrorState from './ErrorState.svelte';
	import LoadingIndicator from './LoadingIndicator.svelte';

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
	$: loadingMinHeight = Math.max(8, Math.min(18, loadingRows * 3));
</script>

<section class="surface min-w-0 overflow-hidden">
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

	{#if loading}
		<div class="flex items-center justify-center px-4 py-6" style={`min-height:${loadingMinHeight}rem`} aria-busy="true">
			<LoadingIndicator label={`Loading ${title || 'table'}`} />
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
