<script lang="ts">
	import { PackageOpen, Plus } from '@lucide/svelte';
	export let title: string;
	export let description = '';
	export let actionLabel = '';
	export let actionHref  = '';
	export let compact = false;
</script>

<div class="flex flex-col items-center justify-center px-6 text-center {compact ? 'py-10' : 'py-16'}">
	<div class="mb-4 flex {compact ? 'h-10 w-10' : 'h-12 w-12'} items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-400 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-500">
		<PackageOpen class="{compact ? 'h-5 w-5' : 'h-6 w-6'}" aria-hidden="true" />
	</div>

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
