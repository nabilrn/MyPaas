<script lang="ts">
	import { Plus } from '@lucide/svelte';
	import ActionLink from './ActionLink.svelte';

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
		<ActionLink href={actionHref} variant="primary" className="mt-4">
			<Plus slot="icon" class="h-4 w-4" />
			{actionLabel}
		</ActionLink>
	{/if}
</div>
