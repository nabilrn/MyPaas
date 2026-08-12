<script lang="ts">
	import { Info } from '@lucide/svelte';
	import { dismissable } from '$lib/actions/dismissable';
	import { infoDisclosureState } from './InfoDisclosure';

	export let label = 'More information';
	export let id = '';

	const disclosure = infoDisclosureState();
	let expanded = disclosure.expanded;
	$: panelId = id || `info-${label.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')}`;

	function syncExpanded() {
		expanded = disclosure.expanded;
	}

	function toggle() {
		disclosure.toggle();
		syncExpanded();
	}

	function close() {
		disclosure.close();
		syncExpanded();
	}
</script>

<span class="relative inline-block align-middle" use:dismissable={{ enabled: expanded, onDismiss: close }}>
	<button
		type="button"
		class="app-focus inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-gray-500 dark:hover:bg-gray-900 dark:hover:text-gray-200"
		aria-label={label}
		aria-expanded={expanded}
		aria-controls={panelId}
		title={label}
		on:click={toggle}
	>
		<Info class="h-4 w-4" aria-hidden="true" />
	</button>
	{#if expanded}
		<span id={panelId} class="overlay absolute left-9 top-0 z-20 w-72 px-3 py-2.5 text-sm leading-5 text-gray-600 dark:text-gray-300">
			<slot />
		</span>
	{/if}
</span>
