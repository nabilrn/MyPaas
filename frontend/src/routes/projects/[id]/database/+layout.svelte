<script lang="ts">
	import { page } from '$app/stores';

	$: base = `/projects/${$page.params.id}/database`;
	$: pathname = $page.url.pathname;
	$: schemaActive = pathname.startsWith(`${base}/schema`);

	function tabClass(active: boolean) {
		return `rounded-md px-3 py-2 text-sm font-medium transition-colors ${active
			? 'bg-gray-950 text-white dark:bg-white dark:text-gray-950'
			: 'text-gray-600 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-300 dark:hover:bg-neutral-900 dark:hover:text-white'}`;
	}
</script>

<nav class="mb-4 inline-flex rounded-lg border border-gray-200 bg-white p-1 dark:border-neutral-800 dark:bg-neutral-950" aria-label="Database Studio views">
	<a href={base} class={tabClass(!schemaActive)} aria-current={!schemaActive ? 'page' : undefined}>Data</a>
	<a href={`${base}/schema`} class={tabClass(schemaActive)} aria-current={schemaActive ? 'page' : undefined}>Schema & ERD</a>
</nav>

<slot />
