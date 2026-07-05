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

	$: baseClass = 'inline-flex items-center justify-center gap-2 whitespace-nowrap font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-gray-950 focus:ring-offset-2 focus:ring-offset-white disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-white dark:focus:ring-offset-gray-950';
	$: sizeClass = {
		xs: 'min-h-8 rounded-md px-2.5 py-1.5 text-xs',
		sm: 'min-h-9 rounded-md px-3 py-1.5 text-sm',
		md: 'min-h-10 rounded-md px-4 py-2 text-sm'
	}[size];
	$: variantClass = {
		primary: 'bg-gray-950 text-white hover:bg-gray-800 dark:bg-white dark:text-gray-950 dark:hover:bg-gray-200',
		secondary: 'border border-gray-300 bg-white text-gray-800 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-900',
		danger: 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500',
		ghost: 'text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100',
		ghostDanger: 'text-gray-500 hover:bg-red-50 hover:text-red-600 focus:ring-red-500 dark:text-gray-400 dark:hover:bg-red-950/30 dark:hover:text-red-300'
	}[variant];
	$: classes = `${baseClass} ${sizeClass} ${variantClass} ${full ? 'w-full' : ''} ${className}`.trim();
</script>

<button
	{type}
	class={classes}
	disabled={disabled || loading}
	aria-busy={loading}
	aria-label={ariaLabel}
	on:click
>
	{#if loading}
		<span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent"></span>
	{/if}
	<span class="min-w-0 truncate">
		{#if loading && loadingLabel}
			{loadingLabel}
		{:else}
			<slot />
		{/if}
	</span>
</button>
