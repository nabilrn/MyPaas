<script lang="ts">
	import { Info } from '@lucide/svelte';
	import { infoDisclosureState } from './InfoDisclosure';

	export let label = 'More information';
	export let id = '';

	const disclosure = infoDisclosureState();
	let expanded = disclosure.expanded;
	$: panelId = id || `info-${label.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')}`;

	function toggle() {
		disclosure.toggle();
		expanded = disclosure.expanded;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key !== 'Escape' || !expanded) return;
		event.preventDefault();
		disclosure.close();
		expanded = disclosure.expanded;
	}
</script>

<span class="relative inline-block align-middle">
	<button
		type="button"
		class="app-focus inline-flex h-6 w-6 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-gray-500 dark:hover:bg-gray-900 dark:hover:text-gray-200"
		aria-label={label}
		aria-expanded={expanded}
		aria-controls={panelId}
		title={label}
		on:click={toggle}
		on:keydown={handleKeydown}
	>
		<Info class="h-3.5 w-3.5" aria-hidden="true" />
	</button>
	{#if expanded}
		<span
			id={panelId}
			class="absolute left-7 top-0 z-20 w-64 rounded-md border border-gray-200 bg-white px-3 py-2 text-xs leading-5 text-gray-600 shadow-lg dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300"
		>
			<slot />
		</span>
	{/if}
</span>
