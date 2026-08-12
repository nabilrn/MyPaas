<script lang="ts">
	import { LoaderCircle } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';

	type IconButtonVariant = 'primary' | 'secondary' | 'ghost' | 'ghostDanger' | 'danger' | 'default' | 'brand';

	export let label: string;
	export let href = '';
	export let type: 'button' | 'submit' | 'reset' = 'button';
	export let variant: IconButtonVariant = 'secondary';
	export let disabled = false;
	export let loading = false;
	export let external = false;
	export let className = '';

	const dispatch = createEventDispatcher<{ click: MouseEvent }>();

	$: normalizedVariant = variant === 'default' || variant === 'brand'
		? 'secondary'
		: variant === 'danger'
			? 'ghostDanger'
			: variant;
	$: variantClass = {
		primary:
			'border-gray-950 bg-gray-950 text-white hover:border-black hover:bg-black dark:border-white dark:bg-white dark:text-gray-950 dark:hover:border-gray-200 dark:hover:bg-gray-200',
		secondary:
			'border-gray-300 bg-white text-gray-700 hover:border-gray-400 hover:bg-gray-50 hover:text-gray-950 dark:border-gray-700 dark:bg-neutral-950 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-neutral-900 dark:hover:text-white',
		ghost:
			'border-transparent bg-transparent text-gray-500 hover:border-gray-200 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:border-gray-800 dark:hover:bg-neutral-900 dark:hover:text-white',
		ghostDanger:
			'border-transparent bg-transparent text-red-600 hover:border-red-100 hover:bg-red-50 hover:text-red-700 focus-visible:ring-red-500 dark:text-red-300 dark:hover:border-red-950/50 dark:hover:bg-red-950/30 dark:hover:text-red-200'
	}[normalizedVariant];

	$: isUnavailable = disabled || loading;
	$: effectiveHref = isUnavailable ? undefined : href;
	$: disabledClass = isUnavailable ? 'cursor-not-allowed opacity-50' : '';
	$: controlClass = `inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-950 focus-visible:ring-offset-2 focus-visible:ring-offset-white active:translate-y-px disabled:translate-y-0 dark:focus-visible:ring-white dark:focus-visible:ring-offset-neutral-950 ${variantClass} ${disabledClass} ${className}`;

	function handleClick(event: MouseEvent) {
		if (isUnavailable) {
			event.preventDefault();
			event.stopPropagation();
			return;
		}
		dispatch('click', event);
	}
</script>

{#if href}
	<a
		href={effectiveHref}
		class={controlClass}
		data-icon-button
		aria-label={label}
		aria-disabled={isUnavailable}
		aria-busy={loading}
		tabindex={isUnavailable ? -1 : undefined}
		title={label}
		target={external && !isUnavailable ? '_blank' : undefined}
		rel={external && !isUnavailable ? 'noopener' : undefined}
		on:click={handleClick}
	>
		{#if loading}
			<LoaderCircle class="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
		{:else}
			<slot />
		{/if}
	</a>
{:else}
	<button {type} class={controlClass} data-icon-button aria-label={label} aria-busy={loading} title={label} disabled={isUnavailable} on:click={handleClick}>
		{#if loading}
			<LoaderCircle class="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
		{:else}
			<slot />
		{/if}
	</button>
{/if}

<style>
	@media (any-pointer: coarse) {
		[data-icon-button] {
			min-width: 44px;
			min-height: 44px;
		}
	}
</style>
