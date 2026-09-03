<script lang="ts">
	import { CheckSquare2, GitBranch, KeyRound, SlidersHorizontal } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import ProjectSecondaryNavItem from './ProjectSecondaryNavItem.svelte';

	type WizardSection = 'source' | 'configuration' | 'environment' | 'review';
	let activeSection: WizardSection = 'source';

	const items = [
		{ id: 'source' as const, label: 'Source', icon: GitBranch },
		{ id: 'configuration' as const, label: 'Configuration', icon: SlidersHorizontal },
		{ id: 'environment' as const, label: 'Environment', icon: KeyRound },
		{ id: 'review' as const, label: 'Review', icon: CheckSquare2 }
	];

	onMount(() => {
		const hash = window.location.hash.slice(1) as WizardSection;
		if (items.some((item) => item.id === hash)) activeSection = hash;
	});

	function heading(text: string) {
		return Array.from(document.querySelectorAll('form h2')).find((node) => node.textContent?.trim() === text)?.closest('section') ?? null;
	}

	function configurationTarget() {
		return heading('Preparing project')
			?? heading('Deployment setup')
			?? Array.from(document.querySelectorAll('form summary')).find((node) => node.textContent?.includes('Advanced settings'))?.closest('section')
			?? null;
	}

	function reviewTarget() {
		return document.querySelector('form button[type="submit"]')?.closest('.border-t') ?? document.querySelector('form button[type="submit"]');
	}

	function targetFor(section: WizardSection) {
		if (section === 'source') return heading('Source');
		if (section === 'configuration') return configurationTarget();
		if (section === 'environment') return heading('Environment');
		return reviewTarget();
	}

	function navigate(event: MouseEvent, section: WizardSection) {
		event.preventDefault();
		activeSection = section;
		window.history.replaceState(null, '', `${window.location.pathname}${window.location.search}#${section}`);
		requestAnimationFrame(() => {
			targetFor(section)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
		});
	}
</script>

<nav aria-label="Create project steps" class="space-y-1 lg:sticky lg:top-4">
	<p class="px-2 pb-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-gray-400 dark:text-gray-500">Create project</p>
	{#each items as item}
		<ProjectSecondaryNavItem
			active={activeSection === item.id}
			href={`#${item.id}`}
			label={item.label}
			icon={item.icon}
			on:click={(event) => navigate(event, item.id)}
		/>
	{/each}
</nav>
