<script lang="ts">
	import ProjectSettingsSection from '$components/ProjectSettingsSection.svelte';
</script>

<div class="project-settings-source">
	<ProjectSettingsSection section="source" />
</div>

<style>
	.project-settings-source :global(.field) {
		max-width: 36rem;
	}

	.project-settings-source :global(button[aria-label='Deployment branch']),
	.project-settings-source :global(button[aria-label='Base directory']),
	.project-settings-source :global(div[role='listbox'][aria-label='Deployment branch']),
	.project-settings-source :global(div[role='listbox'][aria-label='Base directory']) {
		max-width: 36rem;
	}
</style>
