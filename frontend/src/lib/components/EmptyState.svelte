<script lang="ts">
	import { Plus } from '@lucide/svelte';
	export let title: string;
	export let description = '';
	export let actionLabel = '';
	export let actionHref  = '';
	export let compact = false;
</script>

<div class="flex flex-col items-center justify-center px-6 text-center {compact ? 'py-8' : 'py-12'}">
	<h3 class="{compact ? 'text-sm' : 'text-base'} font-semibold text-gray-900 dark:text-white">{title}</h3>

	{#if description}
		<p class="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">{description}</p>
	{/if}

	{#if actionLabel && actionHref}
		<a
			href={actionHref}
			class="mt-4 inline-flex min-h-9 items-center gap-2 rounded-md bg-brand-700 px-3 py-1.5 text-sm font-medium
				   text-white transition-colors hover:bg-brand-900 focus-visible:outline-none focus-visible:ring-2
				   focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white
				   dark:bg-brand-500 dark:text-gray-950 dark:hover:bg-brand-100 dark:focus-visible:ring-brand-500
				   dark:focus-visible:ring-offset-gray-950"
		>
			<Plus class="h-4 w-4" aria-hidden="true" />
			{actionLabel}
		</a>
	{/if}
</div>
