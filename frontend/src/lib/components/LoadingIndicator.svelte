<script lang="ts">
	import BrandLogo from './BrandLogo.svelte';

	export let label = 'Loading';
	export let size: 'sm' | 'md' | 'lg' = 'md';

	$: shellSize = size === 'sm' ? 'h-11 w-11' : size === 'lg' ? 'h-16 w-16' : 'h-14 w-14';
	$: logoSize = size === 'sm' ? '!h-[14px]' : size === 'lg' ? '!h-[19px]' : '!h-[17px]';
</script>

<div class="flex flex-col items-center justify-center gap-3" role="status" aria-live="polite" aria-label={label}>
	<div class={`relative flex ${shellSize} items-center justify-center`}>
		<div class="absolute inset-0 rounded-full border border-gray-200 dark:border-neutral-800"></div>
		<div class="absolute inset-0 animate-spin rounded-full border-2 border-transparent border-t-gray-950 motion-reduce:animate-none dark:border-t-white"></div>
		<div class="flex h-[62%] w-[62%] items-center justify-center rounded-lg border border-gray-200 bg-white dark:border-neutral-800 dark:bg-neutral-950">
			<BrandLogo compact imageClass={logoSize} />
		</div>
	</div>
	<span class="sr-only">{label}</span>
</div>
