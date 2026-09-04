<script lang="ts">
	import { CheckSquare2, GitBranch, KeyRound, SlidersHorizontal } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import ProjectSecondaryNavItem from './ProjectSecondaryNavItem.svelte';

	type SectionId = 'source' | 'environment' | 'advanced' | 'create';
	let activeSection: SectionId = 'source';

	const items: Array<{ id: SectionId; label: string; icon: any }> = [
		{ id: 'source', label: 'Source', icon: GitBranch },
		{ id: 'environment', label: 'Environment', icon: KeyRound },
		{ id: 'advanced', label: 'Advanced', icon: SlidersHorizontal },
		{ id: 'create', label: 'Create', icon: CheckSquare2 }
	];

	function heading(text: string) {
		return Array.from(document.querySelectorAll('form h2')).find((node) => node.textContent?.trim() === text)?.closest('section') ?? null;
	}

	function targetFor(section: SectionId): Element | null {
		if (section === 'source') return heading('Source');
		if (section === 'environment') return heading('Environment');
		if (section === 'advanced') {
			return Array.from(document.querySelectorAll('form summary')).find((node) => node.textContent?.includes('Advanced settings'))?.closest('section') ?? null;
		}
		return document.querySelector('form button[type="submit"]')?.closest('form > div:last-child')
			?? document.querySelector('form button[type="submit"]');
	}

	function navigate(event: MouseEvent, section: SectionId) {
		event.preventDefault();
		activeSection = section;
		if (typeof window !== 'undefined') {
			window.history.replaceState(null, '', `${window.location.pathname}${window.location.search}#${section}`);
			requestAnimationFrame(() => targetFor(section)?.scrollIntoView({ behavior: 'smooth', block: 'start' }));
		}
	}

	onMount(() => {
		const requested = window.location.hash.slice(1) as SectionId;
		if (items.some((item) => item.id === requested)) activeSection = requested;
	});
</script>

<nav aria-label="New project sections" class="space-y-1 lg:sticky lg:top-4">
	<p class="px-2 pb-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-gray-400 dark:text-gray-500">New project</p>
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
