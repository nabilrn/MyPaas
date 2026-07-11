<script lang="ts">
	export let title = '';
	export let description = '';
	export let className = '';
</script>

<header class={`mb-6 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between ${className}`.trim()}>
	<div class="min-w-0">
		<h1 class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{title}</h1>
		{#if description}
			<p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">{description}</p>
		{/if}
		{#if $$slots.meta}
			<div class="mt-3 flex flex-wrap items-center gap-2">
				<slot name="meta" />
			</div>
		{/if}
	</div>

	{#if $$slots.actions}
		<div class="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-center">
			<slot name="actions" />
		</div>
	{/if}
</header>
