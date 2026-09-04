<script lang="ts">
	import { CheckSquare2, GitBranch, KeyRound, SlidersHorizontal } from '@lucide/svelte';
	import ProjectSecondaryNavItem from './ProjectSecondaryNavItem.svelte';
	import { createProjectWizard, setCreateProjectStep, type CreateProjectStep } from '$lib/stores/create-project-wizard';

	const items: Array<{ id: CreateProjectStep; label: string; icon: any }> = [
		{ id: 'source', label: 'Source', icon: GitBranch },
		{ id: 'configuration', label: 'Configuration', icon: SlidersHorizontal },
		{ id: 'environment', label: 'Environment', icon: KeyRound },
		{ id: 'review', label: 'Review', icon: CheckSquare2 }
	];

	function canOpen(step: CreateProjectStep) {
		if (step === 'source') return true;
		if (step === 'configuration') return $createProjectWizard.sourceComplete;
		if (step === 'environment') return $createProjectWizard.sourceComplete && $createProjectWizard.configurationComplete;
		return $createProjectWizard.sourceComplete
			&& $createProjectWizard.configurationComplete
			&& $createProjectWizard.environmentComplete;
	}

	function navigate(event: MouseEvent, step: CreateProjectStep) {
		event.preventDefault();
		if (!canOpen(step) || $createProjectWizard.busy) return;
		setCreateProjectStep(step);
		if (typeof window !== 'undefined') {
			window.history.replaceState(null, '', `${window.location.pathname}${window.location.search}#${step}`);
		}
	}
</script>

<nav aria-label="Create project steps" class="space-y-1 lg:sticky lg:top-4">
	<p class="px-2 pb-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-gray-400 dark:text-gray-500">Create project</p>
	{#each items as item}
		<ProjectSecondaryNavItem
			active={$createProjectWizard.activeStep === item.id}
			disabled={!canOpen(item.id) || $createProjectWizard.busy}
			href={`#${item.id}`}
			label={item.label}
			icon={item.icon}
			on:click={(event) => navigate(event, item.id)}
		/>
	{/each}
</nav>
