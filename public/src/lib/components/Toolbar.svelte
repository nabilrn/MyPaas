<script lang="ts">
	export let className = '';
</script>

<div class={`soft-panel flex flex-col gap-3 p-3 sm:flex-row sm:items-center sm:justify-between ${className}`.trim()}>
	<div class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
		<slot />
	</div>

	{#if $$slots.actions}
		<div class="flex shrink-0 flex-wrap items-center gap-2">
			<slot name="actions" />
		</div>
	{/if}
</div>
