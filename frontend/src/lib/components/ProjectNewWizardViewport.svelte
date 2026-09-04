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
	const stepCopy: Record<CreateProjectStep, { title: string; description: string }> = {
		source: { title: 'Source', description: 'Choose what MyPaaS should inspect and deploy.' },
		configuration: { title: 'Configuration', description: 'Review detected runtime settings and change only what needs an override.' },
		environment: { title: 'Environment', description: 'Complete required variables and add any application-specific values.' },
		review: { title: 'Review', description: 'Confirm the generated deployment configuration before creating the project.' }
	};

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
		for (const section of sections) {
			section.dataset.createProjectStep = classifySection(section);
			if (section.dataset.createProjectStep === 'configuration') {
				const disclosure = section.querySelector('details');
				if (disclosure) disclosure.open = true;
			}
		}

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

	$: currentCopy = stepCopy[$createProjectWizard.activeStep];
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

{#if $createProjectWizard.activeStep === 'configuration'}
	<header class="create-project-step-header border-b border-[color:var(--workspace-divider)] px-5 py-4">
		<h1 class="text-sm font-semibold text-gray-950 dark:text-white">{currentCopy.title}</h1>
		<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{currentCopy.description}</p>
	</header>
{/if}

<div bind:this={root} class="create-project-wizard min-w-0" data-active-create-step={$createProjectWizard.activeStep}>
	<slot />
</div>

{#if $createProjectWizard.activeStep !== 'review'}
	<div class="create-project-wizard-footer flex items-center justify-between gap-3 border-t border-[color:var(--workspace-divider)] px-4 py-3">
		<p class="min-w-0 truncate text-xs text-gray-500 dark:text-gray-400">
			{#if $createProjectWizard.activeStep === 'source'}Choose a source and let MyPaaS finish analysis before continuing.{/if}
			{#if $createProjectWizard.activeStep === 'configuration'}Detected values are safe defaults. Override them only when the repository needs it.{/if}
			{#if $createProjectWizard.activeStep === 'environment'}Required values can remain incomplete until Review, where blockers stay visible.{/if}
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
	.create-project-step-header,
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

	/* GitHub selection is the primary Git-source action; manual URL remains the explicit fallback. */
	:global(.create-project-wizard section[data-create-project-step='source'] div:has(> div > #repo)) {
		display: flex !important;
		flex-direction: column;
		align-items: stretch !important;
		gap: 0.75rem !important;
	}

	:global(.create-project-wizard section[data-create-project-step='source'] div:has(> div > #repo) > button) {
		order: -1;
		min-height: 3rem;
		width: 100%;
		justify-content: center;
	}

	:global(.create-project-wizard section[data-create-project-step='source'] div:has(> div > #repo) > div) {
		width: 100%;
	}

	/* In wizard mode Advanced settings becomes the Configuration panel, not a nested card. */
	:global(.create-project-wizard section[data-create-project-step='configuration']) {
		padding: 0 !important;
	}

	:global(.create-project-wizard section[data-create-project-step='configuration'] > details) {
		border: 0;
		border-radius: 0;
		background: transparent;
	}

	:global(.create-project-wizard section[data-create-project-step='configuration'] > details > summary) {
		display: none;
	}

	:global(.create-project-wizard section[data-create-project-step='configuration'] > details > div) {
		border: 0;
		border-radius: 0;
		background: transparent;
		padding-top: 1rem;
	}

	@media (max-width: 639px) {
		.create-project-wizard-footer {
			align-items: flex-end;
		}
	}
</style>
