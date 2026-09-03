<script lang="ts">
	import ProjectSettingsSection from '$components/ProjectSettingsSection.svelte';
</script>

<div class="project-settings-source">
	<ProjectSettingsSection section="source" />
</div>

<style>
	.project-settings-source :global(.space-y-5 > :first-child) {
		padding-inline: 1.25rem;
	}

	.project-settings-source :global(.space-y-5 > .border-t),
	.project-settings-source :global(.space-y-5 > .divide-y > .grid),
	.project-settings-source :global(.space-y-5 > details),
	.project-settings-source :global(.space-y-5 > .flex:last-child) {
		padding-inline: 1rem;
	}

	.project-settings-source :global(.space-y-5 > :not([hidden]) ~ :not([hidden])) {
		margin-top: 0.75rem;
	}

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
