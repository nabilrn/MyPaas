<script lang="ts">
	import ProjectNewSidebar from '$components/ProjectNewSidebar.svelte';
</script>

<div class="grid min-h-[calc(100vh-3.5rem)] lg:grid-cols-[12rem_minmax(0,1fr)]">
	<aside class="border-b border-[color:var(--workspace-divider)] px-3 py-4 lg:border-b-0 lg:border-r">
		<ProjectNewSidebar />
	</aside>

	<main class="min-w-0 px-3.5 py-3">
		<div class="new-project-content min-w-0">
			<slot />
		</div>
	</main>
</div>

<style>
	:global(.new-project-content > .page-shell) {
		width: 100%;
		max-width: none;
		margin-inline: 0;
		padding: 0;
	}
</style>
