<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import ActionButton from './ActionButton.svelte';
	import {
		createProjectWizard,
		setCreateProjectStep,
		updateCreateProjectWizard,
		type CreateProjectStep
	} from '$lib/stores/create-project-wizard';

	let root: HTMLDivElement;
	let observer: MutationObserver | undefined;
	let frame = 0;

	const order: CreateProjectStep[] = ['source', 'configuration', 'environment', 'review'];

	function sectionHeading(section: HTMLElement) {
		return section.querySelector('h2')?.textContent?.trim() ?? '';
	}

	function classifySection(section: HTMLElement): CreateProjectStep {
		const heading = sectionHeading(section);
		const summary = section.querySelector('summary')?.textContent?.trim() ?? '';
		if (heading === 'Source' || heading === 'Preparing project') return 'source';
		if (heading === 'Environment') return 'environment';
		if (heading === 'Deployment setup') return 'review';
		if (summary.includes('Advanced settings')) return 'configuration';
		if (section.querySelector('button[type="submit"]')) return 'review';
		if (section.textContent?.includes('Agent assistance')) return 'review';
		return 'configuration';
	}

	function syncSections() {
		if (!root) return;
		const form = root.querySelector('form');
		if (!form) return;
		const sections = Array.from(form.children).filter((node): node is HTMLElement => node instanceof HTMLElement && node.tagName === 'SECTION');
		for (const section of sections) section.dataset.createProjectStep = classifySection(section);

		const setupReady = sections.some((section) => sectionHeading(section) === 'Deployment setup');
		const submit = form.querySelector('button[type="submit"]') as HTMLButtonElement | null;
		const busy = Boolean(form.querySelector('[aria-busy="true"]'));

		if (!setupReady && $createProjectWizard.activeStep !== 'source') setCreateProjectStep('source');
		updateCreateProjectWizard({
			sourceComplete: setupReady,
			configurationComplete: setupReady,
			environmentComplete: setupReady,
			reviewReady: Boolean(submit && !submit.disabled),
			busy
		});
		applyVisibility();
	}

	function applyVisibility() {
		if (!root) return;
		for (const section of Array.from(root.querySelectorAll<HTMLElement>('form > section[data-create-project-step]'))) {
			section.hidden = section.dataset.createProjectStep !== $createProjectWizard.activeStep;
		}
	}

	function scheduleSync() {
		cancelAnimationFrame(frame);
		frame = requestAnimationFrame(syncSections);
	}

	function move(step: number) {
		const current = order.indexOf($createProjectWizard.activeStep);
		const target = order[Math.max(0, Math.min(order.length - 1, current + step))];
		if (!target) return;
		if (target !== 'source' && !$createProjectWizard.sourceComplete) return;
		setCreateProjectStep(target);
		if (typeof window !== 'undefined') window.history.replaceState(null, '', `${window.location.pathname}${window.location.search}#${target}`);
	}

	$: if (root && $createProjectWizard.activeStep) applyVisibility();

	onMount(() => {
		const requested = window.location.hash.slice(1) as CreateProjectStep;
		if (order.includes(requested)) setCreateProjectStep(requested);
		observer = new MutationObserver(scheduleSync);
		observer.observe(root, {
			childList: true,
			subtree: true,
			attributes: true,
			attributeFilter: ['disabled', 'aria-busy']
		});
		scheduleSync();
	});

	onDestroy(() => {
		observer?.disconnect();
		cancelAnimationFrame(frame);
	});
</script>

<div bind:this={root} class="create-project-wizard min-w-0">
	<slot />
</div>

{#if $createProjectWizard.activeStep !== 'review'}
	<div class="create-project-wizard-footer flex items-center justify-between gap-3 border-t border-[color:var(--workspace-divider)] px-4 py-3">
		<p class="min-w-0 truncate text-xs text-gray-500 dark:text-gray-400">
			{#if $createProjectWizard.activeStep === 'source'}Choose a source and let MyPaaS finish analysis before continuing.{/if}
			{#if $createProjectWizard.activeStep === 'configuration'}Review detected runtime settings. Change only what needs an override.{/if}
			{#if $createProjectWizard.activeStep === 'environment'}Fill required values, import an .env file, or add variables manually.{/if}
		</p>
		<div class="flex shrink-0 gap-2">
			{#if $createProjectWizard.activeStep !== 'source'}
				<ActionButton type="button" variant="secondary" on:click={() => move(-1)}>Back</ActionButton>
			{/if}
			<ActionButton
				type="button"
				disabled={$createProjectWizard.activeStep === 'source' && (!$createProjectWizard.sourceComplete || $createProjectWizard.busy)}
				on:click={() => move(1)}
			>
				Continue
			</ActionButton>
		</div>
	</div>
{/if}

<style>
	.create-project-wizard,
	.create-project-wizard-footer {
		width: min(100%, 64rem);
	}

	:global(.create-project-wizard > .page-shell) {
		width: 100%;
		max-width: none;
		margin: 0;
		padding: 0;
	}

	:global(.create-project-wizard form.surface) {
		border: 0;
		border-radius: 0;
		background: transparent;
		box-shadow: none;
	}

	:global(.create-project-wizard form > section) {
		border-color: var(--workspace-divider) !important;
	}

	:global(.create-project-wizard form > section[hidden]) {
		display: none !important;
	}

	:global(.create-project-wizard section[data-create-project-step='configuration'] > details) {
		border-left: 0;
		border-right: 0;
		border-radius: 0;
		background: transparent;
	}

	:global(.create-project-wizard section[data-create-project-step='configuration'] > details > summary),
	:global(.create-project-wizard section[data-create-project-step='configuration'] > details > div) {
		border-radius: 0;
		background: transparent;
	}

	@media (max-width: 639px) {
		.create-project-wizard-footer {
			align-items: flex-end;
		}
	}
</style>
