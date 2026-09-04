<script lang="ts">
	import ProjectNewSidebar from '$components/ProjectNewSidebar.svelte';
</script>

<div class="grid min-h-[calc(100vh-3.5rem)] lg:grid-cols-[12rem_minmax(0,1fr)]">
	<aside class="border-b border-[color:var(--workspace-divider)] px-3 py-4 lg:border-b-0 lg:border-r">
		<ProjectNewSidebar />
	</aside>

	<main class="min-w-0 px-3.5 py-3">
		<div class="new-project-content min-w-0 w-full">
			<slot />
		</div>
	</main>
</div>

<style>
	:global(.new-project-content > .page-shell) {
		width: 100%;
		max-width: none;
		margin: 0;
		padding: 0;
	}

	:global(.new-project-content form.surface) {
		width: 100%;
		max-width: none;
		overflow: visible;
		border: 0;
		border-radius: 0;
		background: transparent;
		box-shadow: none;
	}

	:global(.new-project-content form > section),
	:global(.new-project-content form > div:last-child) {
		border-color: var(--workspace-divider) !important;
	}

	:global(.new-project-content form > section) {
		scroll-margin-top: 4.5rem;
	}

	/* Selection changes stroke/state only; Create Project does not introduce a second filled-card hierarchy. */
	:global(.new-project-content [role='group'] button[aria-pressed='true']) {
		border-color: var(--app-border-strong) !important;
		background: transparent !important;
	}

	:global(.new-project-content [role='group'] button[aria-pressed='false']) {
		border-color: var(--control-border) !important;
		background: transparent !important;
	}

	/* First-level setup/assistance blocks stay flat inside the shared workspace surface. */
	:global(.new-project-content form > section > div.rounded-md.border) {
		border-right: 0 !important;
		border-left: 0 !important;
		border-radius: 0 !important;
		background: transparent !important;
	}

	:global(.new-project-content form > section > details.group) {
		border-right: 0 !important;
		border-left: 0 !important;
		border-radius: 0 !important;
		background: transparent !important;
	}

	:global(.new-project-content form > section > div.rounded-md.border .grid.gap-px) {
		background: var(--workspace-divider) !important;
	}

	:global(.new-project-content form > section > div.rounded-md.border .grid.gap-px > div) {
		background: var(--app-surface) !important;
	}

	/* Keep the real readiness state and primary action available while the long form scrolls. */
	:global(.new-project-content form > div:last-child) {
		position: sticky;
		bottom: 0;
		z-index: 20;
		background: var(--app-surface) !important;
	}
</style>
