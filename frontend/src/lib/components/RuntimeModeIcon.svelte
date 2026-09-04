<script lang="ts">
	export let mode: 'static' | 'dockerfile' | 'compose' | 'image' | string;
	export let className = 'h-4 w-4';
</script>

<span class={`inline-flex shrink-0 items-center justify-center text-current ${className}`} aria-hidden="true">
	{#if mode === 'static'}
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" class="h-full w-full">
			<circle cx="12" cy="12" r="8.5" />
			<path d="M3.8 12h16.4M12 3.5c2.2 2.3 3.4 5.1 3.4 8.5S14.2 18.2 12 20.5M12 3.5C9.8 5.8 8.6 8.6 8.6 12S9.8 18.2 12 20.5" />
		</svg>
	{:else if mode === 'compose'}
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" class="h-full w-full">
			<path d="M8 4.5 12 2l4 2.5-4 2.5-4-2.5Z" />
			<path d="M8 4.5V9l4 2.5L16 9V4.5" />
			<path d="m4.5 11 4-2.5 4 2.5-4 2.5-4-2.5Z" />
			<path d="M4.5 11v4.5l4 2.5 4-2.5V11" />
			<path d="m11.5 13 4-2.5 4 2.5-4 2.5-4-2.5Z" />
			<path d="M11.5 13v4.5l4 2.5 4-2.5V13" />
		</svg>
	{:else}
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" class="h-full w-full">
			<path d="M4 10.5h13.2c1.2 0 2.1-.3 2.8-.9-.3 2.8-2.2 6.4-7.7 6.4H7.2c-2.3 0-3.8-1.2-4.2-3.5" />
			<path d="M6 8.5h2.6V6H6v2.5Zm3.5 0h2.6V6H9.5v2.5Zm3.5 0h2.6V6H13v2.5ZM9.5 5h2.6V2.5H9.5V5Z" />
			<path d="M20 8.3c.8-.5 1.3-1.2 1.5-2.2 1 .8 1.4 1.8 1.2 3" />
		</svg>
	{/if}
</span>
