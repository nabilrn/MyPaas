<script lang="ts">
	export let section: 'general' | 'source' | 'resources' | 'webhook' | 'danger' = 'general';
</script>

<div class="settings-workspace-contract" data-settings-section={section}>
	<slot />
</div>

<style>
	.settings-workspace-contract {
		width: 100%;
		max-width: 64rem;
	}

	/* Shared geometry contract for form-heavy settings pages:
	   - one 64rem content measure
	   - 1.25rem title gutter aligned with the established project leaves
	   - 1rem body/control gutter
	   - one full-width divider between structural blocks
	   - inset borders only for alerts/controls, not for page structure
	   - decorative icons are not part of ordinary field rows */
	.settings-workspace-contract :global(.space-y-4),
	.settings-workspace-contract :global(.space-y-5) {
		width: 100%;
	}

	.settings-workspace-contract :global(.space-y-4 > :not([hidden]) ~ :not([hidden])),
	.settings-workspace-contract :global(.space-y-5 > :not([hidden]) ~ :not([hidden])) {
		margin-top: 0;
	}

	.settings-workspace-contract :global(.space-y-4 > :first-child),
	.settings-workspace-contract :global(.space-y-5 > :first-child) {
		border-bottom: 1px solid var(--workspace-divider);
		padding-inline: 1.25rem;
		padding-top: 1rem;
		padding-bottom: 0.75rem;
	}

	.settings-workspace-contract :global(.space-y-5 > .border-t),
	.settings-workspace-contract :global(.space-y-5 > .border-y),
	.settings-workspace-contract :global(.space-y-5 > details),
	.settings-workspace-contract :global(.space-y-4 > section.border-y) {
		border-top-width: 0 !important;
		border-bottom: 1px solid var(--workspace-divider) !important;
	}

	.settings-workspace-contract :global(.space-y-5 > .border-t),
	.settings-workspace-contract :global(.space-y-5 > .border-y:not(.divide-y)),
	.settings-workspace-contract :global(.space-y-5 > details) {
		padding-block: 0.875rem;
		padding-inline: 1rem;
	}

	.settings-workspace-contract :global(.space-y-4 > section.border-y > .divide-y > .grid),
	.settings-workspace-contract :global(.space-y-5 > .divide-y > .grid) {
		padding-inline: 1rem;
	}

	.settings-workspace-contract :global(.space-y-5 > .divide-y),
	.settings-workspace-contract :global(.space-y-4 > section.border-y .divide-y) {
		border-color: var(--workspace-divider) !important;
	}

	.settings-workspace-contract :global(.space-y-5 > .flex.border-b) {
		border-color: var(--workspace-divider) !important;
		padding-block: 0.75rem;
		padding-inline: 1rem;
	}

	.settings-workspace-contract :global(.space-y-4 > .flex:last-child),
	.settings-workspace-contract :global(.space-y-5 > .flex:last-child) {
		border-bottom: 1px solid var(--workspace-divider);
		padding-block: 0.75rem;
		padding-inline: 1rem;
	}

	.settings-workspace-contract :global(.space-y-5 > .alert-danger),
	.settings-workspace-contract :global(.space-y-5 > .alert-neutral) {
		margin: 0.75rem 1rem !important;
	}

	.settings-workspace-contract[data-settings-section='source'] :global(.field),
	.settings-workspace-contract[data-settings-section='source'] :global(button[aria-label='Deployment branch']),
	.settings-workspace-contract[data-settings-section='source'] :global(button[aria-label='Base directory']),
	.settings-workspace-contract[data-settings-section='source'] :global(div[role='listbox'][aria-label='Deployment branch']),
	.settings-workspace-contract[data-settings-section='source'] :global(div[role='listbox'][aria-label='Base directory']) {
		max-width: 36rem;
	}

	.settings-workspace-contract[data-settings-section='resources'] :global(button[aria-label='Resource profile']),
	.settings-workspace-contract[data-settings-section='resources'] :global(div[role='listbox'][aria-label='Resource profile']),
	.settings-workspace-contract[data-settings-section='resources'] :global(.border-y > .mt-4.grid) {
		max-width: 32rem;
	}

	.settings-workspace-contract[data-settings-section='resources'] :global(input.field) {
		max-width: 20rem;
	}

	.settings-workspace-contract[data-settings-section='resources'] :global(textarea.field) {
		max-width: 48rem;
	}

	.settings-workspace-contract[data-settings-section='danger'] :global(.space-y-5 > section.border-y) {
		padding: 1rem;
	}

	.settings-workspace-contract[data-settings-section='danger'] :global(input.field) {
		max-width: 36rem;
	}
</style>
