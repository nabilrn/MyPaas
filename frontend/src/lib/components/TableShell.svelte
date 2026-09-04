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
	export let contentClass = '';

	const dispatch = createEventDispatcher<{ retry: void }>();
</script>

<section class="surface workspace-section min-w-0 overflow-hidden !rounded-none !border-0" aria-busy={loading}>
	{#if title || description}
		<div class="panel-header">
			<div class="min-w-0">
				{#if title}
					<h2 class="panel-title">{title}</h2>
				{/if}
				{#if description}
					<p class="panel-description">{description}</p>
				{/if}
			</div>
		</div>
	{/if}

	{#if $$slots.actions}
		<div class="table-toolbar">
			<slot name="actions" />
		</div>
	{/if}

	{#if error && !loading}
		<ErrorState message={error} on:retry={() => dispatch('retry')} />
	{:else if empty && !loading}
		<EmptyState title={emptyTitle} description={emptyDescription} compact />
	{:else}
		<slot name="notice" />
		<div class={`table-scroll-region min-w-0 ${contentClass}`.trim()}>
			<slot />
		</div>
		<slot name="footer" />
	{/if}
</section>

<style>
	.table-scroll-region {
		overflow-x: auto !important;
		overflow-y: hidden;
		overscroll-behavior-x: contain;
		-webkit-overflow-scrolling: touch;
	}

	:global(.table-toolbar [data-action-button]),
	:global(.table-toolbar [data-action-link]) {
		min-height: 2.25rem;
	}

	@media (any-pointer: coarse) {
		:global(.table-toolbar [data-action-button]),
		:global(.table-toolbar [data-action-link]) {
			min-height: 44px;
		}
	}
</style>
