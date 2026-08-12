<script lang="ts">
	import { Check } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';

	type SegmentedOption = {
		value: string;
		label: string;
		description?: string;
		disabled?: boolean;
	};

	export let value = '';
	export let options: SegmentedOption[] = [];
	export let label = '';

	const dispatch = createEventDispatcher<{ change: string }>();

	function choose(option: SegmentedOption) {
		if (option.disabled) return;
		value = option.value;
		dispatch('change', option.value);
	}
</script>

<div class="grid gap-2 [grid-template-columns:repeat(auto-fit,minmax(10rem,1fr))]" role="group" aria-label={label || undefined}>
	{#each options as option}
		{@const selected = option.value === value}
		<button
			type="button"
			on:click={() => choose(option)}
			disabled={option.disabled}
			aria-pressed={selected}
			class="app-focus relative min-h-14 rounded-md border px-3 py-2.5 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-50
				{selected
					? 'border-gray-950 bg-gray-50 text-gray-950 dark:border-white dark:bg-neutral-900 dark:text-white'
					: 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-neutral-800 dark:bg-neutral-950 dark:text-gray-300 dark:hover:border-gray-700 dark:hover:bg-neutral-900'}"
		>
			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0">
					<span class="block text-sm font-medium">{option.label}</span>
					{#if option.description}<span class="mt-0.5 block text-xs leading-4 text-gray-500 dark:text-gray-400">{option.description}</span>{/if}
				</div>
				{#if selected}<Check class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />{/if}
			</div>
		</button>
	{/each}
</div>
