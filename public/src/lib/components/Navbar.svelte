<script lang="ts">
	import { ChevronLeft, ChevronRight, ClipboardList, FolderKanban, LogOut, Menu, Moon, Plus, Sun, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import IconButton from '$components/IconButton.svelte';
	import { api } from '$api';
	import { sidebarCollapsed } from '$stores/sidebar';
	import { theme } from '$stores/theme';
	import type { User } from '$types';
	import logoGreen from '../../assets/mypaas-horizontal-transparent-green.png';
	import logoWhite from '../../assets/mypaas-horizontal-transparent-white.png';

	export let user: User | null = null;

	let menuOpen = false;
	let signingOut = false;

	const navItems = [
		{ href: '/projects', label: 'Projects', icon: FolderKanban },
		{ href: '/projects/new', label: 'New project', icon: Plus },
		{ href: '/admin/users', label: 'Users', icon: Users },
		{ href: '/admin/audit-logs', label: 'Audit', icon: ClipboardList }
	];

	$: pathname = $page.url.pathname;

	function isActive(href: string, currentPath = pathname) {
		if (href === '/projects/new') return currentPath === href;
		if (href === '/projects') return currentPath === '/projects' || (currentPath.startsWith('/projects/') && currentPath !== '/projects/new');
		return currentPath === href || currentPath.startsWith(`${href}/`);
	}

	function navItemClass(href: string, currentPath = pathname, collapsed = false) {
		const base = `group relative flex min-h-10 items-center rounded-md border text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950 ${collapsed ? 'justify-center px-0' : 'gap-3 px-3'}`;
		const active = 'border-brand-500/35 bg-brand-50 text-brand-900 dark:border-brand-500/35 dark:bg-brand-500/10 dark:text-brand-100';
		const idle = 'border-transparent text-gray-600 hover:border-gray-200 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:border-gray-800 dark:hover:bg-gray-900 dark:hover:text-white';
		return `${base} ${isActive(href, currentPath) ? active : idle}`;
	}

	async function handleLogout() {
		if (signingOut) return;
		signingOut = true;
		try {
			await api.auth.logout();
		} finally {
			await goto('/login');
		}
	}
</script>

<nav class="sticky top-0 z-40 border-b border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950 lg:hidden">
	<div class="flex h-14 items-center justify-between px-4">
		<a href="/projects" class="flex min-w-0 items-center">
			<span class="sr-only">MyPaas</span>
			<img src={logoGreen} alt="" aria-hidden="true" class="h-8 w-[122px] object-contain object-left dark:hidden" />
			<img src={logoWhite} alt="" aria-hidden="true" class="hidden h-8 w-[122px] object-contain object-left dark:block" />
		</a>
		<button
			type="button"
			on:click={() => (menuOpen = !menuOpen)}
			class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-gray-200 text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:border-gray-800 dark:text-gray-400 dark:hover:bg-gray-900 dark:hover:text-white dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950"
			aria-label="Toggle navigation"
			aria-expanded={menuOpen}
		>
			<Menu class="h-4 w-4" aria-hidden="true" />
		</button>
	</div>
	{#if menuOpen}
		<div class="border-t border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950">
			<div class="grid gap-1">
				{#each navItems as item}
					<a
						href={item.href}
						on:click={() => (menuOpen = false)}
						aria-current={isActive(item.href, pathname) ? 'page' : undefined}
						class={navItemClass(item.href, pathname)}
					>
						<svelte:component this={item.icon} class="h-4 w-4 shrink-0" aria-hidden="true" />
						{item.label}
					</a>
				{/each}
			</div>
		</div>
	{/if}
</nav>

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

	<div class="flex-1 overflow-y-auto py-4 {$sidebarCollapsed ? 'px-2' : 'px-3'}">
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
	</div>

	<div class="border-t border-gray-200 dark:border-gray-800 {$sidebarCollapsed ? 'p-2' : 'p-3'}">
		{#if $sidebarCollapsed}
			<div class="flex flex-col items-center gap-2">
				<IconButton label="Toggle dark mode" variant="brand" on:click={() => theme.toggle()}>
					{#if $theme === 'dark'}
						<Sun class="h-4 w-4" aria-hidden="true" />
					{:else}
						<Moon class="h-4 w-4" aria-hidden="true" />
					{/if}
				</IconButton>
				<IconButton label="Sign out" variant="danger" loading={signingOut} on:click={handleLogout}>
					<LogOut class="h-4 w-4" aria-hidden="true" />
				</IconButton>
			</div>
		{:else}
		<div class="mb-3 flex items-center justify-between gap-2 rounded-md border border-gray-200 bg-gray-50 p-2 dark:border-gray-800 dark:bg-gray-900">
			<div class="flex min-w-0 items-center gap-2">
				{#if user?.avatarUrl}
					<img src={user.avatarUrl} alt={user.githubUsername ?? user.email ?? 'User'} class="h-8 w-8 rounded-full object-cover" />
				{:else}
					<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gray-200 text-xs font-semibold text-gray-600 dark:bg-gray-800 dark:text-gray-300">
						{(user?.email || '?').slice(0, 1).toUpperCase()}
					</span>
				{/if}
				<div class="min-w-0">
					<p class="truncate text-sm font-medium text-gray-950 dark:text-white">{user?.githubUsername ?? user?.email}</p>
					<p class="truncate text-xs text-gray-500 dark:text-gray-400">{user?.email}</p>
				</div>
			</div>
			<button
				on:click={() => theme.toggle()}
				class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-brand-100 bg-brand-50 text-brand-700 transition-colors hover:border-brand-500/40 hover:bg-brand-100 hover:text-brand-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-gray-50 dark:border-brand-500/35 dark:bg-brand-500/10 dark:text-brand-500 dark:hover:border-brand-500/50 dark:hover:bg-brand-500/15 dark:hover:text-brand-100 dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-900"
				aria-label="Toggle dark mode"
			>
				{#if $theme === 'dark'}
					<Sun class="h-4 w-4" aria-hidden="true" />
				{:else}
					<Moon class="h-4 w-4" aria-hidden="true" />
				{/if}
			</button>
		</div>
		<ActionButton
			variant="ghostDanger"
			size="xs"
			full
			on:click={handleLogout}
			loading={signingOut}
			loadingLabel="Signing out..."
			className="justify-start"
		>
			Sign out
		</ActionButton>
		{/if}
	</div>
</aside>
