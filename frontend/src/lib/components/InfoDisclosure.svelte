<script lang="ts">
	import { Info } from '@lucide/svelte';
	import { infoDisclosureState } from './InfoDisclosure';

	export let label = 'More information';
	export let id = `info-${Math.random().toString(36).slice(2, 9)}`;

	const disclosure = infoDisclosureState();
	let expanded = disclosure.expanded;

	function syncExpanded(open: boolean) {
		if (open) disclosure.open();
		else disclosure.close();
		expanded = disclosure.expanded;
	}

	function handleToggle(event: Event) {
		syncExpanded((event.currentTarget as HTMLDetailsElement).open);
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key !== 'Escape' || !expanded) return;
		event.preventDefault();
		disclosure.close();
		expanded = disclosure.expanded;
	}
</script>

<details class="relative inline-block align-middle" open={expanded} on:toggle={handleToggle} on:keydown={handleKeydown}>
	<summary
		class="app-focus inline-flex h-6 w-6 cursor-pointer list-none items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-gray-500 dark:hover:bg-gray-900 dark:hover:text-gray-200 [&::-webkit-details-marker]:hidden"
		aria-label={label}
		aria-expanded={expanded}
		aria-controls={id}
		title={label}
	>
		<Info class="h-3.5 w-3.5" aria-hidden="true" />
	</summary>
	<span
		{id}
		class="absolute left-7 top-0 z-20 w-64 rounded-md border border-gray-200 bg-white px-3 py-2 text-xs leading-5 text-gray-600 shadow-lg dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300"
	>
		<slot />
	</span>
</details>
