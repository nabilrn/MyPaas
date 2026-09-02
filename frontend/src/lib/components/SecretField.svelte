<script lang="ts">
	import { Copy, Eye, EyeOff, Pencil, RotateCcw, Trash2, X } from '@lucide/svelte';
	import { afterUpdate, createEventDispatcher } from 'svelte';
	import ActionButton from './ActionButton.svelte';
	import IconButton from './IconButton.svelte';

	export let keyName = '';
	export let value = '';
	export let revealed = false;
	export let dirty = false;
	export let revealing = false;
	export let deleting = false;
	export let stateLabel = '';

	let editing = false;
	let previousDirty = dirty;

	const dispatch = createEventDispatcher<{
		change: string;
		copy: void;
		discard: void;
		reveal: void;
		remove: void;
	}>();

	afterUpdate(() => {
		if (previousDirty && !dirty) editing = false;
		previousDirty = dirty;
	});

	function cancelEdit() {
		if (dirty) dispatch('discard');
		editing = false;
	}
</script>

<div class="grid gap-3 px-4 py-2.5 lg:grid-cols-[14rem_minmax(0,1fr)_17rem] lg:items-center">
	<div class="min-w-0">
		<div class="flex min-w-0 items-center gap-2">
			{#if dirty}<span class="status-dot bg-amber-500" aria-label="Unsaved change"></span>{/if}
			<p class="truncate font-mono text-sm font-medium text-gray-950 dark:text-white">{keyName}</p>
		</div>
		<p class="mt-0.5 text-xs {dirty ? 'text-amber-700 dark:text-amber-200' : 'text-gray-500 dark:text-gray-400'}">{stateLabel}</p>
	</div>

	{#if editing}
		<input
			type={revealed ? 'text' : 'password'}
			{value}
			placeholder="Enter replacement value"
			autocomplete="new-password"
			spellcheck="false"
			on:input={(event) => dispatch('change', (event.currentTarget as HTMLInputElement).value)}
			class="field w-full font-mono"
			aria-label={`Replacement value for ${keyName}`}
		/>
	{:else}
		<div class="field flex min-h-9 w-full items-center font-mono text-sm text-gray-700 dark:text-gray-300" aria-label={`${keyName} stored value`}>
			<span class="truncate">{revealed && value ? value : '••••••••'}</span>
		</div>
	{/if}

	<div class="flex flex-wrap items-center gap-1.5 lg:justify-end">
		{#if editing}
			<ActionButton variant="ghost" size="xs" on:click={cancelEdit} disabled={revealing || deleting}>
				{#if dirty}<RotateCcw slot="icon" class="h-3.5 w-3.5" />{:else}<X slot="icon" class="h-3.5 w-3.5" />{/if}
				{dirty ? 'Discard' : 'Cancel'}
			</ActionButton>
		{:else}
			<ActionButton variant="ghost" size="xs" on:click={() => (editing = true)} disabled={revealing || deleting}>
				<Pencil slot="icon" class="h-3.5 w-3.5" />
				Edit
			</ActionButton>
		{/if}

		{#if revealed && value && !editing}
			<IconButton label={`Copy ${keyName}`} variant="ghost" on:click={() => dispatch('copy')} disabled={revealing || deleting}>
				<Copy class="h-4 w-4" aria-hidden="true" />
			</IconButton>
		{/if}
		<IconButton label={revealed ? `Hide ${keyName}` : `Reveal ${keyName}`} variant="ghost" on:click={() => dispatch('reveal')} loading={revealing} disabled={deleting || editing}>
			{#if revealed}
				<EyeOff class="h-4 w-4" aria-hidden="true" />
			{:else}
				<Eye class="h-4 w-4" aria-hidden="true" />
			{/if}
		</IconButton>
		<ActionButton variant="ghostDanger" size="xs" on:click={() => dispatch('remove')} loading={deleting} loadingLabel="Deleting" disabled={revealing || editing}>
			<Trash2 slot="icon" class="h-3.5 w-3.5" />
			Delete
		</ActionButton>
	</div>
</div>
