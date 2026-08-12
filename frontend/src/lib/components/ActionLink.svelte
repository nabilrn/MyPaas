<script lang="ts">
	export let href: string;
	export let variant: 'primary' | 'secondary' | 'ghost' = 'secondary';
	export let size: 'xs' | 'sm' | 'md' = 'sm';
	export let full = false;
	export let external = false;
	export let ariaLabel: string | undefined = undefined;
	export let className = '';

	$: baseClass =
		'inline-flex min-w-0 items-center justify-center gap-2 whitespace-nowrap font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white active:translate-y-px dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950';
	$: sizeClass = {
		xs: 'min-h-8 rounded-md px-2.5 py-1.5 text-xs',
		sm: 'min-h-9 rounded-md px-3 py-1.5 text-sm',
		md: 'min-h-10 rounded-md px-4 py-2 text-sm'
	}[size];
	$: variantClass = {
		primary: 'bg-brand-700 text-white hover:bg-brand-900 dark:bg-brand-500 dark:text-gray-950 dark:hover:bg-brand-100',
		secondary:
			'border border-gray-300 bg-white text-gray-800 hover:border-gray-400 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-950/80 dark:text-gray-200 dark:hover:border-gray-600 dark:hover:bg-gray-900',
		ghost: 'text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100'
	}[variant];
	$: classes = `${baseClass} ${sizeClass} ${variantClass} ${full ? 'w-full' : ''} ${className}`.trim();
</script>

<a
	{href}
	class={classes}
	data-action-link
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
