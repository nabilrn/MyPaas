<script lang="ts">
	import ProjectNewSidebar from '$components/ProjectNewSidebar.svelte';
	import ProjectNewWizardViewport from '$components/ProjectNewWizardViewport.svelte';
</script>

<div class="grid min-h-[calc(100vh-3.5rem)] lg:grid-cols-[12rem_minmax(0,1fr)]">
	<aside class="border-b border-[color:var(--workspace-divider)] px-3 py-4 lg:border-b-0 lg:border-r">
		<ProjectNewSidebar />
	</aside>

	<main class="min-w-0 px-3.5 py-3">
		<ProjectNewWizardViewport>
			<slot />
		</ProjectNewWizardViewport>
	</main>
</div>
