<script lang="ts">
	import { ChevronLeft, ChevronRight, ClipboardList, FolderKanban, Settings, Users } from '@lucide/svelte';
	import { page } from '$app/stores';
	import IconButton from '$components/IconButton.svelte';
	import { sidebarCollapsed } from '$stores/sidebar';
	import logoGreen from '../../assets/mypaas-horizontal-transparent-green.png';
	import logoWhite from '../../assets/mypaas-horizontal-transparent-white.png';

	const navItems = [
		{ href: '/projects', label: 'Projects', icon: FolderKanban },
		{ href: '/admin/users', label: 'Users', icon: Users },
		{ href: '/admin/audit-logs', label: 'Audit', icon: ClipboardList },
		{ href: '/admin/settings', label: 'Settings', icon: Settings }
	];

	$: pathname = $page.url.pathname;

	function isActive(href: string, currentPath = pathname) {
		if (href === '/projects') return currentPath === '/projects' || currentPath.startsWith('/projects/');
		return currentPath === href || currentPath.startsWith(`${href}/`);
	}

	function navItemClass(href: string, currentPath = pathname, collapsed = false) {
		const base = `group relative flex min-h-10 items-center rounded-md border text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950 ${collapsed ? 'justify-center px-0' : 'gap-3 px-3'}`;
		const active = 'border-brand-500/35 bg-brand-50 text-brand-900 dark:border-brand-500/35 dark:bg-brand-500/10 dark:text-brand-100';
		const idle = 'border-transparent text-gray-600 hover:border-gray-200 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:border-gray-800 dark:hover:bg-gray-900 dark:hover:text-white';
		return `${base} ${isActive(href, currentPath) ? active : idle}`;
	}
</script>

<aside class="fixed inset-y-0 left-0 z-40 hidden flex-col border-r border-gray-200 bg-white transition-[width] duration-200 dark:border-gray-800 dark:bg-gray-950 lg:flex {$sidebarCollapsed ? 'w-16' : 'w-64'}">
	<div class="flex h-16 items-center border-b border-gray-200 dark:border-gray-800 {$sidebarCollapsed ? 'justify-center px-2' : 'justify-between gap-2.5 px-4'}">
		{#if !$sidebarCollapsed}
			<a href="/projects" class="flex min-w-0 items-center">
				<span class="sr-only">MyPaas</span>
				<img src={logoGreen} alt="" aria-hidden="true" class="h-9 w-[138px] object-contain object-left dark:hidden" />
				<img src={logoWhite} alt="" aria-hidden="true" class="hidden h-9 w-[138px] object-contain object-left dark:block" />
			</a>
		{/if}
		<IconButton
			label={$sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
			variant={$sidebarCollapsed ? 'ghost' : 'default'}
			on:click={() => sidebarCollapsed.toggle()}
		>
			{#if $sidebarCollapsed}
				<ChevronRight class="h-4 w-4" aria-hidden="true" />
			{:else}
				<ChevronLeft class="h-4 w-4" aria-hidden="true" />
			{/if}
		</IconButton>
	</div>

	<nav class="flex-1 overflow-y-auto py-4 {$sidebarCollapsed ? 'px-2' : 'px-3'}" aria-label="Primary navigation">
		{#if !$sidebarCollapsed}
			<p class="px-3 pb-2 text-[11px] font-medium text-gray-400 dark:text-gray-500">Workspace</p>
		{/if}
		<div class="space-y-1">
			{#each navItems as item}
				<a
					href={item.href}
					aria-current={isActive(item.href, pathname) ? 'page' : undefined}
					class={navItemClass(item.href, pathname, $sidebarCollapsed)}
					title={$sidebarCollapsed ? item.label : undefined}
				>
					<svelte:component this={item.icon} class="h-4 w-4 shrink-0" aria-hidden="true" />
					{#if $sidebarCollapsed}
						<span class="sr-only">{item.label}</span>
					{:else}
						{item.label}
					{/if}
				</a>
			{/each}
		</div>
	</nav>
</aside>
