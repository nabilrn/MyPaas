<script lang="ts">
	import { page } from '$app/stores';
	import ProjectEnvironmentSettings from '$components/ProjectEnvironmentSettings.svelte';
</script>

<svelte:head>
	<title>Environment · MyPaaS</title>
</svelte:head>

<div class="project-environment-leaf w-full space-y-3">
	<div class="px-5 pt-4">
		<h1 class="text-lg font-semibold text-gray-950 dark:text-white">Environment</h1>
		<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Variables available to your app at runtime.</p>
	</div>
	<ProjectEnvironmentSettings projectId={$page.params.id ?? ''} />
</div>

<style>
	.project-environment-leaf :global(.border-t > .flex:first-child) {
		padding-inline: 1rem;
	}

	:global(.project-environment-leaf > .border-t > .flex > p:first-child) {
		display: none;
	}
</style>
