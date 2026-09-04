<script lang="ts">
	export let active = false;
	export let danger = false;
	export let disabled = false;
	export let href = '';
	export let label = '';
	export let icon: any;
</script>

<a
	{href}
	aria-current={active ? 'page' : undefined}
	aria-disabled={disabled ? 'true' : undefined}
	tabindex={disabled ? -1 : undefined}
	class="app-focus flex w-full items-center gap-2.5 border-l-2 bg-transparent px-2.5 py-2 text-left text-[13px] font-medium transition-colors {disabled
		? 'cursor-not-allowed border-transparent text-gray-300 dark:text-gray-700'
		: danger
			? active
				? 'border-red-600 text-red-700 dark:border-red-400 dark:text-red-300'
				: 'border-transparent text-gray-600 hover:text-red-700 dark:text-gray-400 dark:hover:text-red-300'
			: active
				? 'border-gray-950 text-gray-950 dark:border-white dark:text-white'
				: 'border-transparent text-gray-600 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white'}"
	on:click
>
	<svelte:component this={icon} class="h-4 w-4 shrink-0" aria-hidden="true" />
	{label}
</a>
