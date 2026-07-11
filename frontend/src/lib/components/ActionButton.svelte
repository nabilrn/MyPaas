<script lang="ts">
	export let type: 'button' | 'submit' | 'reset' = 'button';
	export let variant: 'primary' | 'secondary' | 'danger' | 'ghost' | 'ghostDanger' = 'secondary';
	export let size: 'xs' | 'sm' | 'md' = 'sm';
	export let loading = false;
	export let disabled = false;
	export let full = false;
	export let loadingLabel = '';
	export let ariaLabel: string | undefined = undefined;
	export let className = '';

	$: baseClass =
		'inline-flex min-w-0 items-center justify-center gap-2 whitespace-nowrap font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white active:translate-y-px disabled:cursor-not-allowed disabled:translate-y-0 disabled:opacity-55 dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950';
	$: sizeClass = {
		xs: 'min-h-8 rounded-md px-2.5 py-1.5 text-xs',
		sm: 'min-h-9 rounded-md px-3 py-1.5 text-sm',
		md: 'min-h-10 rounded-md px-4 py-2 text-sm'
	}[size];
	$: variantClass = {
		primary: 'bg-brand-700 text-white hover:bg-brand-900 dark:bg-brand-500 dark:text-gray-950 dark:hover:bg-brand-100',
		secondary:
			'border border-gray-300 bg-white text-gray-800 hover:border-gray-400 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-950/80 dark:text-gray-200 dark:hover:border-gray-600 dark:hover:bg-gray-900',
		danger: 'bg-red-600 text-white hover:bg-red-700 focus-visible:ring-red-500 dark:bg-red-500 dark:text-white dark:hover:bg-red-400',
		ghost: 'text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100',
		ghostDanger: 'text-red-600 hover:bg-red-50 hover:text-red-700 focus-visible:ring-red-500 dark:text-red-300 dark:hover:bg-red-950/30 dark:hover:text-red-200'
	}[variant];
	$: classes = `${baseClass} ${sizeClass} ${variantClass} ${full ? 'w-full' : ''} ${className}`.trim();
</script>

<button {type} class={classes} data-action-button disabled={disabled || loading} aria-busy={loading} aria-label={ariaLabel} on:click>
	{#if loading}
		<span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true"></span>
	{/if}
	<span class="min-w-0 truncate">
		{#if loading && loadingLabel}
			{loadingLabel}
		{:else}
			<slot />
		{/if}
	</span>
</button>

<style>
	@media (any-pointer: coarse) {
		[data-action-button] {
			min-height: 44px;
		}
	}
</style>
