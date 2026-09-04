<main class="min-h-[calc(100vh-3.5rem)] min-w-0 px-3.5 py-3">
	<div class="new-project-content min-w-0 w-full">
		<slot />
	</div>
</main>

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

	:global(.new-project-content form > section) {
		border: 0 !important;
		background: transparent !important;
		scroll-margin-top: 4.5rem;
	}

	:global(.new-project-content form > section + section) {
		border-top: 1px solid var(--workspace-divider) !important;
	}

	/* Selection changes stroke and typography only. Never introduce a filled neutral card. */
	:global(.new-project-content [role='group'] button[aria-pressed='true']) {
		border-color: var(--app-border-strong) !important;
		background: transparent !important;
	}

	:global(.new-project-content [role='group'] button[aria-pressed='false']) {
		border-color: var(--control-border) !important;
		background: transparent !important;
	}

	/* First-level setup surfaces stay on the same application surface. */
	:global(.new-project-content form > section > div.rounded-md.border),
	:global(.new-project-content form > section > details.group),
	:global(.new-project-content form > section > details.group > div) {
		border-right: 0 !important;
		border-left: 0 !important;
		border-radius: 0 !important;
		background: transparent !important;
		box-shadow: none !important;
	}

	:global(.new-project-content form > section > div.rounded-md.border .grid.gap-px) {
		background: transparent !important;
		gap: 0 !important;
	}

	:global(.new-project-content form > section > div.rounded-md.border .grid.gap-px > div) {
		background: transparent !important;
	}

	/* Neutral hover/selected fills are not hierarchy in Create Project. */
	:global(.new-project-content form button[class~='bg-gray-50']),
	:global(.new-project-content form button[class~='bg-gray-100']),
	:global(.new-project-content form button[class~='dark:bg-neutral-800']),
	:global(.new-project-content form button[class~='dark:bg-neutral-900']) {
		background: transparent !important;
	}

	/* Keep one readiness/action bar visible while the long form scrolls. */
	:global(.new-project-content form > div:last-child) {
		position: sticky;
		bottom: 0;
		z-index: 20;
		border-top: 1px solid var(--workspace-divider) !important;
		background: var(--app-surface) !important;
	}
</style>
