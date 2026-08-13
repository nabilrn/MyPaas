<script lang="ts">
	import { ChevronLeft, ClipboardList, FolderKanban, LogOut, Moon, Settings, Sun, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import BrandLogo from '$components/BrandLogo.svelte';
	import IconButton from '$components/IconButton.svelte';
	import { api } from '$api';
	import { dismissable } from '$lib/actions/dismissable';
	import { sidebarCollapsed } from '$stores/sidebar';
	import { theme } from '$stores/theme';
	import type { User } from '$types';

	export let user: User | null = null;

	const navItems = [
		{ href: '/projects', label: 'Projects', icon: FolderKanban },
		{ href: '/admin/users', label: 'Users', icon: Users },
		{ href: '/admin/audit-logs', label: 'Audit', icon: ClipboardList },
		{ href: '/admin/settings', label: 'Settings', icon: Settings }
	];

	let accountMenuOpen = false;
	let signingOut = false;

	$: pathname = $page.url.pathname;
	$: userLabel = user?.githubUsername ?? user?.email ?? 'Account';

	function isActive(href: string, currentPath = pathname) {
		if (href === '/projects') return currentPath === '/projects' || currentPath.startsWith('/projects/');
		return currentPath === href || currentPath.startsWith(`${href}/`);
	}

	function navItemClass(href: string, currentPath = pathname, collapsed = false) {
		const base = `group relative flex min-h-10 items-center rounded-md border text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-950 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-white dark:focus-visible:ring-offset-neutral-950 ${collapsed ? 'justify-center px-0' : 'gap-3 px-3'}`;
		const active = 'border-gray-200 bg-gray-100 text-gray-950 dark:border-neutral-800 dark:bg-neutral-900 dark:text-white';
		const idle = 'border-transparent text-gray-600 hover:border-gray-200 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:border-neutral-800 dark:hover:bg-neutral-900 dark:hover:text-white';
		return `${base} ${isActive(href, currentPath) ? active : idle}`;
	}

	function initial() {
		return (user?.githubUsername || user?.email || '?').slice(0, 1).toUpperCase();
	}

	function closeAccountMenu() {
		accountMenuOpen = false;
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

<aside class="fixed inset-y-0 left-0 z-40 hidden flex-col border-r border-gray-200 bg-white transition-[width] duration-200 dark:border-neutral-800 dark:bg-neutral-950 lg:flex {$sidebarCollapsed ? 'w-16' : 'w-64'}">
	<div class="flex h-16 items-center border-b border-gray-200 dark:border-neutral-800 {$sidebarCollapsed ? 'justify-center px-2' : 'justify-between gap-2.5 px-5'}">
		{#if $sidebarCollapsed}
			<button
				type="button"
				class="app-focus flex h-10 w-10 items-center justify-center rounded-md border border-transparent transition-colors hover:border-gray-200 hover:bg-gray-100 dark:hover:border-neutral-800 dark:hover:bg-neutral-900"
				aria-label="Expand sidebar"
				title="Expand sidebar"
				on:click={() => sidebarCollapsed.toggle()}
			>
				<BrandLogo compact />
			</button>
		{:else}
			<a href="/projects" class="flex min-w-0 items-center rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-950 dark:focus-visible:ring-white">
				<BrandLogo />
			</a>
			<IconButton label="Collapse sidebar" variant="ghost" on:click={() => sidebarCollapsed.toggle()}>
				<ChevronLeft class="h-4 w-4" aria-hidden="true" />
			</IconButton>
		{/if}
	</div>

	<nav class="flex-1 overflow-y-auto py-4 {$sidebarCollapsed ? 'px-2' : 'px-3'}" aria-label="Primary navigation">
		{#if !$sidebarCollapsed}<p class="px-3 pb-2 text-xs font-medium text-gray-400 dark:text-gray-500">Workspace</p>{/if}
		<div class="space-y-1">
			{#each navItems as item}
				<a href={item.href} aria-current={isActive(item.href, pathname) ? 'page' : undefined} class={navItemClass(item.href, pathname, $sidebarCollapsed)} title={$sidebarCollapsed ? item.label : undefined}>
					<svelte:component this={item.icon} class="h-4 w-4 shrink-0" aria-hidden="true" />
					{#if $sidebarCollapsed}<span class="sr-only">{item.label}</span>{:else}{item.label}{/if}
				</a>
			{/each}
		</div>
	</nav>

	<div class="border-t border-gray-200 p-2 dark:border-neutral-800">
		<div class="relative" use:dismissable={{ enabled: accountMenuOpen, onDismiss: closeAccountMenu }}>
			{#if $sidebarCollapsed}
				<button type="button" class="app-focus mx-auto flex h-9 w-9 items-center justify-center overflow-hidden rounded-md border border-transparent text-xs font-semibold text-gray-700 transition-colors hover:border-gray-200 hover:bg-gray-100 dark:text-gray-200 dark:hover:border-neutral-800 dark:hover:bg-neutral-900" aria-label="Open account menu" aria-expanded={accountMenuOpen} title={userLabel} on:click={() => (accountMenuOpen = !accountMenuOpen)}>
					{#if user?.avatarUrl}<img src={user.avatarUrl} alt="" class="h-7 w-7 rounded-full object-cover" />{:else}<span class="flex h-7 w-7 items-center justify-center rounded-full bg-gray-100 dark:bg-neutral-800">{initial()}</span>{/if}
				</button>
			{:else}
				<button type="button" class="app-focus flex w-full items-center gap-3 rounded-md border border-transparent px-2 py-2 text-left transition-colors hover:border-gray-200 hover:bg-gray-100 dark:hover:border-neutral-800 dark:hover:bg-neutral-900" aria-label="Open account menu" aria-expanded={accountMenuOpen} on:click={() => (accountMenuOpen = !accountMenuOpen)}>
					{#if user?.avatarUrl}<img src={user.avatarUrl} alt="" class="h-8 w-8 shrink-0 rounded-full object-cover" />{:else}<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-700 dark:bg-neutral-800 dark:text-gray-200">{initial()}</span>{/if}
					<span class="min-w-0 flex-1"><span class="block truncate text-sm font-medium text-gray-950 dark:text-white">{userLabel}</span>{#if user?.email}<span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400">{user.email}</span>{/if}</span>
				</button>
			{/if}
			{#if accountMenuOpen}
				<div class="overlay absolute z-50 mb-2 w-56 p-1 {$sidebarCollapsed ? 'bottom-0 left-full ml-2' : 'bottom-full left-0 right-0 w-auto'}">
					<ActionButton variant="ghostDanger" size="xs" full className="justify-start" loading={signingOut} loadingLabel="Signing out..." on:click={handleLogout}><LogOut slot="icon" class="h-4 w-4" />Sign out</ActionButton>
				</div>
			{/if}
		</div>

		{#if $sidebarCollapsed}
			<div class="mt-1 flex justify-center"><IconButton label={$theme === 'dark' ? 'Use light appearance' : 'Use dark appearance'} variant="ghost" on:click={() => theme.toggle()}>{#if $theme === 'dark'}<Sun class="h-4 w-4" aria-hidden="true" />{:else}<Moon class="h-4 w-4" aria-hidden="true" />{/if}</IconButton></div>
		{:else}
			<ActionButton variant="ghost" size="xs" full className="mt-1 justify-start" on:click={() => theme.toggle()}>{#if $theme === 'dark'}<Sun slot="icon" class="h-4 w-4" />Light appearance{:else}<Moon slot="icon" class="h-4 w-4" />Dark appearance{/if}</ActionButton>
		{/if}
	</div>
</aside>