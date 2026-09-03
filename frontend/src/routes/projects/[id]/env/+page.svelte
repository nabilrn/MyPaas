<script lang="ts">
	import { page } from '$app/stores';
	import ProjectEnvironmentSettings from '$components/ProjectEnvironmentSettings.svelte';
</script>

<svelte:head>
	<title>Environment · MyPaaS</title>
</svelte:head>

<div class="w-full space-y-4">
	<div>
		<h1 class="text-lg font-semibold text-gray-950 dark:text-white">Environment</h1>
		<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Variables available to your app at runtime.</p>
	</div>
	<ProjectEnvironmentSettings projectId={$page.params.id ?? ''} />
</div>
