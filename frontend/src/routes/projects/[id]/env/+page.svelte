<script lang="ts">
	import { page } from '$app/stores';
	import ProjectEnvironmentSettings from '$components/ProjectEnvironmentSettings.svelte';
</script>

<svelte:head>
	<title>Environment · MyPaaS</title>
</svelte:head>

<ProjectEnvironmentSettings projectId={$page.params.id ?? ''} />
