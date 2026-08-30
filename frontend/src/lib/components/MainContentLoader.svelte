<script lang="ts">
	import LoadingIndicator from './LoadingIndicator.svelte';

	export let label = 'Loading';
</script>

<div
	class="absolute inset-0 z-30 min-h-[20rem] bg-white/95 backdrop-blur-[1px] dark:bg-neutral-950/95"
	role="presentation"
>
	<div class="sticky top-0 flex min-h-[calc(100vh-4rem)] items-center justify-center">
		<LoadingIndicator {label} size="lg" />
	</div>
</div>
