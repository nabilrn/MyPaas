<script lang="ts">
	import BrandLogo from '$components/BrandLogo.svelte';

	export let label = 'Loading';
</script>

<div
	class="absolute inset-0 z-30 min-h-[20rem] bg-white/92 backdrop-blur-[1px] dark:bg-neutral-950/92"
	role="status"
	aria-live="polite"
	aria-label={label}
>
	<div class="sticky top-0 flex min-h-[calc(100vh-4rem)] items-center justify-center">
		<div class="flex flex-col items-center gap-3">
			<div class="relative flex h-16 w-16 items-center justify-center">
				<div class="absolute inset-0 rounded-full border border-gray-200 dark:border-neutral-800"></div>
				<div class="absolute inset-0 animate-spin rounded-full border-2 border-transparent border-t-gray-950 motion-reduce:animate-none dark:border-t-white"></div>
				<div class="flex h-10 w-10 items-center justify-center rounded-xl border border-gray-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
					<BrandLogo compact imageClass="!h-[19px]" />
				</div>
			</div>
			<span class="sr-only">{label}</span>
		</div>
	</div>
</div>
