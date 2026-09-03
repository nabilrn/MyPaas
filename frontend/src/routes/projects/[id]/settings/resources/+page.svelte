<script lang="ts">
	import ProjectSettingsSection from '$components/ProjectSettingsSection.svelte';
</script>

<div class="project-settings-resources">
	<ProjectSettingsSection section="resources" />
</div>

<style>
	.project-settings-resources :global(.space-y-5 > :not([hidden]) ~ :not([hidden])) {
		margin-top: 0.75rem;
	}

	.project-settings-resources :global(button[aria-label='Resource profile']),
	.project-settings-resources :global(div[role='listbox'][aria-label='Resource profile']) {
		max-width: 32rem;
	}

	.project-settings-resources :global(input.field) {
		max-width: 20rem;
	}

	.project-settings-resources :global(textarea.field) {
		max-width: 48rem;
	}

	.project-settings-resources :global(.border-y > .mt-4.grid) {
		max-width: 32rem;
	}
</style>
