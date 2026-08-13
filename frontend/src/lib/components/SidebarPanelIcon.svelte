<script lang="ts">
	export let collapsed = false;
	export let className = 'h-4 w-4';
</script>

<svg
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="1.6"
	stroke-linecap="round"
	stroke-linejoin="round"
	class={className}
	aria-hidden="true"
>
	<rect x="3" y="4" width="18" height="16" rx="4" />
	{#if collapsed}
		<rect x="14" y="4.8" width="6.2" height="14.4" rx="3" fill="currentColor" stroke="none" opacity="0.08" />
		<path d="M14 4.5v15" />
	{:else}
		<rect x="3.8" y="4.8" width="6.2" height="14.4" rx="3" fill="currentColor" stroke="none" opacity="0.08" />
		<path d="M10 4.5v15" />
	{/if}
</svg>
