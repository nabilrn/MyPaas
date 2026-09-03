<script lang="ts">
	import { Check, ChevronDown } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';

	export type SelectMenuOption = {
		value: string;
		label: string;
		description?: string;
	};

	export let value = '';
	export let options: SelectMenuOption[] = [];
	export let ariaLabel = 'Select option';
	export let disabled = false;
	export let placeholder = 'Select';

	let open = false;

	const dispatch = createEventDispatcher<{ change: string }>();

	$: selected = options.find((option) => option.value === value);

	function choose(nextValue: string) {
		if (disabled) return;
		value = nextValue;
		open = false;
		dispatch('change', nextValue);
	}

	function handleWindowKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') open = false;
	}
</script>

<svelte:window on:keydown={handleWindowKeydown} on:click={() => (open = false)} />

<div class="relative" on:click|stopPropagation>
	<button
		type="button"
		class="app-focus flex h-9 w-full items-center justify-between gap-3 rounded-md border border-gray-200 bg-transparent px-3 text-left text-sm text-gray-900 transition-[border-color,color] hover:border-gray-400 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-800 dark:text-gray-100 dark:hover:border-neutral-600"
		aria-label={ariaLabel}
		aria-haspopup="listbox"
		aria-expanded={open}
		{disabled}
		on:click={() => (open = !open)}
	>
		<span class="min-w-0 flex-1 truncate">{selected?.label ?? placeholder}</span>
		<ChevronDown class="h-4 w-4 shrink-0 text-gray-400 transition-transform" class:rotate-180={open} aria-hidden="true" />
	</button>

	{#if open}
		<div class="absolute z-40 mt-1 max-h-64 w-full overflow-y-auto rounded-md border border-gray-200 bg-white p-1 shadow-lg dark:border-neutral-800 dark:bg-neutral-950" role="listbox" aria-label={ariaLabel}>
			{#each options as option}
				<button
					type="button"
					class="app-focus flex w-full items-start gap-2 rounded px-2.5 py-2 text-left text-sm text-gray-700 transition-[border-color,color] hover:text-gray-950 dark:text-gray-300 dark:hover:text-white {option.value === value ? 'border-l-2 border-gray-900 pl-2 font-medium text-gray-950 dark:border-white dark:text-white' : 'border-l-2 border-transparent'}"
					role="option"
					aria-selected={option.value === value}
					on:click={() => choose(option.value)}
				>
					<span class="min-w-0 flex-1">
						<span class="block truncate">{option.label}</span>
						{#if option.description}<span class="mt-0.5 block truncate text-xs font-normal text-gray-500 dark:text-gray-400">{option.description}</span>{/if}
					</span>
					{#if option.value === value}<Check class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>
