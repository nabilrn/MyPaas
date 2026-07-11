<script lang="ts">
	import { Copy, Eye, EyeOff, Trash2 } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';
	import ActionButton from './ActionButton.svelte';
	import IconButton from './IconButton.svelte';

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

<div class="grid gap-3 px-5 py-3 lg:grid-cols-[14rem_minmax(0,1fr)_14rem] lg:items-center">
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

	<div class="flex flex-wrap items-center gap-1 lg:justify-end">
		{#if dirty}
			<ActionButton variant="ghost" size="xs" on:click={() => dispatch('discard')} disabled={revealing || deleting}>Discard</ActionButton>
		{/if}
		{#if value}
			<IconButton label={`Copy ${keyName}`} variant="ghost" on:click={() => dispatch('copy')} disabled={revealing || deleting}>
				<Copy class="h-4 w-4" aria-hidden="true" />
			</IconButton>
		{/if}
		<IconButton label={revealed ? `Hide ${keyName}` : `Reveal ${keyName}`} variant="ghost" on:click={() => dispatch('reveal')} loading={revealing} disabled={deleting}>
			{#if revealed}
				<EyeOff class="h-4 w-4" aria-hidden="true" />
			{:else}
				<Eye class="h-4 w-4" aria-hidden="true" />
			{/if}
		</IconButton>
		<IconButton label={`Delete ${keyName}`} variant="danger" on:click={() => dispatch('remove')} loading={deleting} disabled={revealing}>
			<Trash2 class="h-4 w-4" aria-hidden="true" />
		</IconButton>
	</div>
</div>
