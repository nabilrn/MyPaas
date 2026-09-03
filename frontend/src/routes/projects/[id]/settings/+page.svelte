<script lang="ts">
	import ProjectSettingsSection from '$components/ProjectSettingsSection.svelte';
</script>

<div class="project-settings-general">
	<ProjectSettingsSection section="general" />
</div>

<style>
	.project-settings-general :global(.space-y-4 > :first-child) {
		padding-inline: 1.25rem;
	}

	.project-settings-general :global(section.border-y > .divide-y > .grid) {
		padding-inline: 1rem;
	}

	.project-settings-general :global(.space-y-4 > :not([hidden]) ~ :not([hidden])) {
		margin-top: 0.75rem;
	}
</style>
