<script lang="ts">
	import { ChevronRight } from '@lucide/svelte';
	type BreadcrumbItem = {
		label: string;
		href?: string;
	};

	export let items: BreadcrumbItem[] = [];
</script>

{#if items.length > 0}
	<nav aria-label="Breadcrumb" class="mb-4 flex min-w-0 items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400">
		{#each items as item, index}
			{#if index > 0}
				<ChevronRight class="h-3.5 w-3.5 shrink-0 text-gray-300 dark:text-gray-700" aria-hidden="true" />
			{/if}
			{#if item.href && index < items.length - 1}
				<a href={item.href} class="truncate font-medium hover:text-gray-950 dark:hover:text-white">
					{item.label}
				</a>
			{:else}
				<span class="truncate font-medium text-gray-800 dark:text-gray-200" aria-current={index === items.length - 1 ? 'page' : undefined}>
					{item.label}
				</span>
			{/if}
		{/each}
	</nav>
{/if}
