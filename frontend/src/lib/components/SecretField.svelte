<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import ActionButton from './ActionButton.svelte';

	export let keyName = '';
	export let value = '';
	export let revealed = false;
	export let dirty = false;
	export let revealing = false;
	export let deleting = false;
	export let stateLabel = '';

	const dispatch = createEventDispatcher<{
		change: string;
		copy: void;
		discard: void;
		reveal: void;
		remove: void;
	}>();
</script>

<div class="grid gap-3 px-5 py-3 lg:grid-cols-[14rem_minmax(0,1fr)_12rem] lg:items-center">
	<div class="min-w-0">
		<p class="truncate font-mono text-sm font-semibold text-gray-950 dark:text-white">
			{keyName}
			{#if dirty}
				<span class="ml-1 text-amber-500" aria-label="Unsaved change">●</span>
			{/if}
		</p>
		<p class="mt-0.5 text-xs {dirty ? 'text-amber-600 dark:text-amber-300' : 'text-gray-500 dark:text-gray-400'}">{stateLabel}</p>
	</div>

	<input
		type={revealed ? 'text' : 'password'}
		{value}
		placeholder="••••••••"
		on:input={(event) => dispatch('change', (event.currentTarget as HTMLInputElement).value)}
		class="field w-full font-mono"
		aria-label={`${keyName} value`}
	/>

	<div class="flex items-center gap-1 lg:justify-end">
		{#if dirty}
			<button
				type="button"
				on:click={() => dispatch('discard')}
				class="inline-flex h-8 items-center rounded-md px-2 text-xs font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white"
			>
				Discard
			</button>
		{/if}
		{#if value}
			<button
				type="button"
				on:click={() => dispatch('copy')}
				class="inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white"
				aria-label={`Copy ${keyName}`}
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M8 8h10v10H8zM6 16H5a2 2 0 01-2-2V5a2 2 0 012-2h9a2 2 0 012 2v1" />
				</svg>
			</button>
		{/if}
		<button
			type="button"
			on:click={() => dispatch('reveal')}
			disabled={revealing}
			class="inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-950 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white"
			aria-label={revealed ? `Hide ${keyName}` : `Reveal ${keyName}`}
		>
			{#if revealing}
				<span class="h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent"></span>
			{:else if revealed}
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M3 3l18 18M10.6 10.6A2 2 0 0013.4 13.4M9.9 4.2A10.8 10.8 0 0112 4c4.5 0 8.3 2.9 9.5 7a10.9 10.9 0 01-3 4.7M6.1 6.1A10.8 10.8 0 002.5 11c1.2 4.1 5 7 9.5 7 1.3 0 2.5-.2 3.6-.7" />
				</svg>
			{:else}
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
					<path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
				</svg>
			{/if}
		</button>
		<ActionButton
			variant="ghostDanger"
			size="xs"
			on:click={() => dispatch('remove')}
			className="px-2"
			loading={deleting}
			ariaLabel={`Delete ${keyName}`}
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
			</svg>
		</ActionButton>
	</div>
</div>
