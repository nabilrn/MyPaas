<script lang="ts">
	export let title: string;
	export let description = '';
	export let actionLabel = '';
	export let actionHref  = '';
</script>

<div class="flex flex-col items-center justify-center px-6 py-16 text-center">
	<div class="mb-4 flex h-12 w-12 items-center justify-center rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900">
		<svg class="h-6 w-6 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
			<path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V7M3 7l9-4 9 4" />
		</svg>
	</div>

	<h3 class="text-base font-semibold text-gray-900 dark:text-white">{title}</h3>

	{#if description}
		<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{description}</p>
	{/if}

	{#if actionLabel && actionHref}
		<a
			href={actionHref}
			class="mt-4 inline-flex items-center gap-2 rounded-md bg-gray-950 px-4 py-2 text-sm font-medium
				   text-white hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-gray-950 focus:ring-offset-2
				   dark:bg-white dark:text-gray-950 dark:hover:bg-gray-200 dark:focus:ring-white dark:focus:ring-offset-gray-950"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
			</svg>
			{actionLabel}
		</a>
	{/if}
</div>
