<script lang="ts">
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

<div
	class="grid gap-2 [grid-template-columns:repeat(auto-fit,minmax(10rem,1fr))]"
	role="group"
	aria-label={label || undefined}
>
	{#each options as option}
		{@const selected = option.value === value}
		<button
			type="button"
			on:click={() => choose(option)}
			disabled={option.disabled}
			aria-pressed={selected}
			class="min-h-16 rounded-md border p-3 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-950 focus-visible:ring-offset-2 focus-visible:ring-offset-white disabled:cursor-not-allowed disabled:opacity-50 dark:focus-visible:ring-white dark:focus-visible:ring-offset-gray-950
				{selected
					? 'border-gray-950 bg-gray-100 text-gray-950 dark:border-white dark:bg-neutral-900 dark:text-white'
					: 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950/80 dark:text-gray-300 dark:hover:border-gray-700 dark:hover:bg-gray-900'}"
		>
			<span class="block text-sm font-semibold">{option.label}</span>
			{#if option.description}
				<span class="mt-1 block text-xs opacity-75">{option.description}</span>
			{/if}
		</button>
	{/each}
</div>
