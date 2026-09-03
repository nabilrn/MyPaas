<script lang="ts">
	import LoadingIndicator from './LoadingIndicator.svelte';

	export let label = 'Loading';
</script>

<!--
	Initial authenticated account/resource loading is owned by main content.
	Normal client-side route navigation keeps the current workspace visible instead of
	painting a blocking transition surface between pages.
-->
<div
	class="absolute inset-0 z-30 min-h-[20rem] bg-[var(--app-surface)]"
	role="presentation"
>
	<div class="sticky top-0 flex min-h-[calc(100vh-4rem)] items-center justify-center">
		<LoadingIndicator {label} size="lg" />
	</div>
</div>
