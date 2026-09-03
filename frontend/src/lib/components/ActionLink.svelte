<script lang="ts">
	export let href: string;
	export let variant: 'primary' | 'secondary' | 'ghost' = 'secondary';
	export let size: 'xs' | 'sm' | 'md' = 'sm';
	export let full = false;
	export let external = false;
	export let ariaLabel: string | undefined = undefined;
	export let className = '';

	$: baseClass =
		'inline-flex h-9 min-h-9 min-w-0 items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-950 focus-visible:ring-offset-2 focus-visible:ring-offset-white active:translate-y-px dark:focus-visible:ring-white dark:focus-visible:ring-offset-neutral-950';
	$: sizeClass = {
		xs: 'px-2.5',
		sm: 'px-3',
		md: 'px-4'
	}[size];
	$: variantClass = {
		primary: 'border border-gray-950 bg-gray-950 text-white hover:border-black hover:bg-black dark:border-white dark:bg-white dark:text-gray-950 dark:hover:border-gray-200 dark:hover:bg-gray-200',
		secondary:
			'border border-gray-300 bg-transparent text-gray-800 hover:border-gray-400 dark:border-gray-700 dark:bg-transparent dark:text-gray-200 dark:hover:border-gray-500',
		ghost: 'text-gray-500 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-neutral-800 dark:hover:text-white'
	}[variant];
	$: classes = `${baseClass} ${sizeClass} ${variantClass} ${full ? 'w-full' : ''} ${className}`.trim();
</script>

<a
	{href}
	class={classes}
	data-action-link
	data-size={size}
	aria-label={ariaLabel}
	target={external ? '_blank' : undefined}
	rel={external ? 'noopener' : undefined}
>
	{#if $$slots.icon}
		<span class="flex shrink-0 items-center" aria-hidden="true"><slot name="icon" /></span>
	{/if}
	<span class="min-w-0 truncate"><slot /></span>
</a>

<style>
	@media (any-pointer: coarse) {
		[data-action-link] {
			min-height: 44px;
		}
	}
</style>
