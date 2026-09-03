<script lang="ts">
	import ProjectSettingsSection from '$components/ProjectSettingsSection.svelte';
</script>

<div class="project-settings-danger">
	<ProjectSettingsSection section="danger" />
</div>

<style>
	.project-settings-danger :global(.space-y-5 > :not([hidden]) ~ :not([hidden])) {
		margin-top: 0.75rem;
	}

	.project-settings-danger :global(input.field) {
		max-width: 36rem;
	}

	.project-settings-danger :global(section) {
		max-width: 56rem;
	}
</style>
