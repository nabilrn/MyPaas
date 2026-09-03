<script lang="ts">
	import ProjectSettingsSection from '$components/ProjectSettingsSection.svelte';
</script>

<div class="project-settings-webhook">
	<ProjectSettingsSection section="webhook" />
</div>

<style>
	.project-settings-webhook :global(.space-y-5 > :not([hidden]) ~ :not([hidden])) {
		margin-top: 0.75rem;
	}

	.project-settings-webhook :global(.divide-y + .flex) {
		border-bottom: 1px solid var(--workspace-divider);
		padding-bottom: 0.75rem;
	}
</style>
