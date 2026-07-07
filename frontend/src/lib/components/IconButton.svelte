<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	export let label: string;
	export let href = '';
	export let type: 'button' | 'submit' | 'reset' = 'button';
	export let variant: 'default' | 'primary' | 'brand' | 'danger' | 'ghost' = 'default';
	export let disabled = false;
	export let loading = false;
	export let external = false;
	export let className = '';

	const dispatch = createEventDispatcher<{ click: MouseEvent }>();

	$: variantClass = {
		default: 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-950 dark:border-gray-800 dark:bg-gray-950/80 dark:text-gray-300 dark:hover:border-gray-700 dark:hover:bg-gray-900 dark:hover:text-white',
		primary: 'border-brand-700 bg-brand-700 text-white hover:border-brand-900 hover:bg-brand-900 dark:border-brand-500 dark:bg-brand-500 dark:text-gray-950 dark:hover:border-brand-100 dark:hover:bg-brand-100',
		brand: 'border-brand-100 bg-brand-50 text-brand-700 hover:border-brand-500/40 hover:bg-brand-100 hover:text-brand-900 dark:border-brand-500/35 dark:bg-brand-500/10 dark:text-brand-500 dark:hover:border-brand-500/50 dark:hover:bg-brand-500/15 dark:hover:text-brand-100',
		danger: 'border-red-200 bg-white text-red-600 hover:border-red-300 hover:bg-red-50 hover:text-red-700 dark:border-red-900/70 dark:bg-gray-950 dark:text-red-300 dark:hover:bg-red-950/30',
		ghost: 'border-transparent bg-transparent text-gray-500 hover:border-gray-200 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:border-gray-800 dark:hover:bg-gray-900 dark:hover:text-white'
	}[variant];

	$: disabledClass = disabled || loading ? 'pointer-events-none opacity-50' : '';
	$: controlClass = `group relative inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950 ${variantClass} ${disabledClass} ${className}`;

	function handleClick(event: MouseEvent) {
		if (disabled || loading) {
			event.preventDefault();
			return;
		}
		dispatch('click', event);
	}
</script>

{#if href}
	<a
		{href}
		class={controlClass}
		aria-label={label}
		aria-disabled={disabled || loading}
		target={external ? '_blank' : undefined}
		rel={external ? 'noopener' : undefined}
		on:click={handleClick}
	>
		{#if loading}
			<span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true"></span>
		{:else}
			<slot />
		{/if}
		<span class="pointer-events-none absolute left-1/2 top-full z-50 mt-2 -translate-x-1/2 whitespace-nowrap rounded-md bg-brand-700 px-2 py-1 text-xs font-medium text-white opacity-0 shadow-sm shadow-brand-900/20 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100 dark:bg-brand-500 dark:text-gray-950">
			{label}
		</span>
	</a>
{:else}
	<button
		{type}
		class={controlClass}
		aria-label={label}
		aria-busy={loading}
		{disabled}
		on:click={handleClick}
	>
		{#if loading}
			<span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true"></span>
		{:else}
			<slot />
		{/if}
		<span class="pointer-events-none absolute left-1/2 top-full z-50 mt-2 -translate-x-1/2 whitespace-nowrap rounded-md bg-brand-700 px-2 py-1 text-xs font-medium text-white opacity-0 shadow-sm shadow-brand-900/20 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100 dark:bg-brand-500 dark:text-gray-950">
			{label}
		</span>
	</button>
{/if}
