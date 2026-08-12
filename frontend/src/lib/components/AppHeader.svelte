<script lang="ts">
	import { Bell, ClipboardList, FolderKanban, Menu, Settings, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from './ActionButton.svelte';
	import IconButton from './IconButton.svelte';
	import { api } from '$api';
	import { theme } from '$stores/theme';
	import type { User } from '$types';

	export let user: User | null = null;

	const navItems = [
		{ href: '/projects', label: 'Projects', icon: FolderKanban },
		{ href: '/admin/users', label: 'Users', icon: Users },
		{ href: '/admin/audit-logs', label: 'Audit', icon: ClipboardList },
		{ href: '/admin/settings', label: 'Settings', icon: Settings }
	];

	let mobileMenuOpen = false;
	let notificationsOpen = false;
	let userMenuOpen = false;
	let signingOut = false;

	$: pathname = $page.url.pathname;
	$: sectionLabel = pathname.startsWith('/admin/users')
		? 'Users'
		: pathname.startsWith('/admin/audit-logs')
			? 'Audit'
			: pathname.startsWith('/admin/settings')
				? 'Settings'
				: 'Projects';
	$: userLabel = user?.githubUsername ?? user?.email ?? 'Account';

	function isActive(href: string) {
		if (href === '/projects') return pathname === '/projects' || pathname.startsWith('/projects/');
		return pathname === href || pathname.startsWith(`${href}/`);
	}

	function navItemClass(href: string) {
		const base = 'flex min-h-10 items-center gap-3 rounded-md border px-3 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950';
		const active = 'border-brand-500/35 bg-brand-50 text-brand-900 dark:border-brand-500/35 dark:bg-brand-500/10 dark:text-brand-100';
		const idle = 'border-transparent text-gray-600 hover:border-gray-200 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:border-gray-800 dark:hover:bg-gray-900 dark:hover:text-white';
		return `${base} ${isActive(href) ? active : idle}`;
	}

	function initial() {
		return (user?.githubUsername || user?.email || '?').slice(0, 1).toUpperCase();
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

<header class="sticky top-0 z-30 border-b border-gray-200 bg-white/95 backdrop-blur dark:border-gray-800 dark:bg-gray-950/95">
	<div class="flex h-14 items-center justify-between gap-3 px-4 lg:h-16 lg:px-6">
		<div class="flex min-w-0 items-center gap-3">
			<IconButton
				label={mobileMenuOpen ? 'Close navigation' : 'Open navigation'}
				variant="ghost"
				className="lg:hidden"
				on:click={() => (mobileMenuOpen = !mobileMenuOpen)}
			>
				<Menu class="h-4 w-4" aria-hidden="true" />
			</IconButton>
			<div class="min-w-0">
				<p class="truncate text-sm font-semibold text-gray-950 dark:text-white">{sectionLabel}</p>
				<p class="hidden text-[11px] text-gray-400 dark:text-gray-500 sm:block">MyPaaS workspace</p>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<div class="relative">
				<IconButton
					label="Notifications"
					variant="ghost"
					on:click={() => {
						notificationsOpen = !notificationsOpen;
						userMenuOpen = false;
					}}
				>
					<Bell class="h-4 w-4" aria-hidden="true" />
				</IconButton>
				{#if notificationsOpen}
					<div class="overlay absolute right-0 mt-2 w-[min(22rem,calc(100vw-2rem))] overflow-hidden">
						<div class="border-b border-gray-100 px-4 py-3 dark:border-gray-800">
							<p class="text-sm font-semibold text-gray-950 dark:text-white">Notifications</p>
							<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Platform updates and operational events will appear here.</p>
						</div>
						<div class="px-4 py-8 text-center">
							<Bell class="mx-auto h-5 w-5 text-gray-300 dark:text-gray-700" aria-hidden="true" />
							<p class="mt-2 text-sm font-medium text-gray-700 dark:text-gray-200">You're all caught up</p>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Release updates, deployment alerts, and resource notifications can use this center later.</p>
						</div>
					</div>
				{/if}
			</div>

			<div class="relative">
				<button
					type="button"
					class="app-focus flex h-8 w-8 items-center justify-center overflow-hidden rounded-full border border-gray-200 bg-gray-100 text-xs font-semibold text-gray-600 transition-colors hover:border-gray-300 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300 dark:hover:border-gray-700"
					aria-label="Open account menu"
					aria-expanded={userMenuOpen}
					title={userLabel}
					on:click={() => {
						userMenuOpen = !userMenuOpen;
						notificationsOpen = false;
					}}
				>
					{#if user?.avatarUrl}
						<img src={user.avatarUrl} alt="" class="h-full w-full object-cover" />
					{:else}
						{initial()}
					{/if}
				</button>
				{#if userMenuOpen}
					<div class="overlay absolute right-0 mt-2 w-64 overflow-hidden p-2">
						<div class="px-2 py-2">
							<p class="truncate text-sm font-semibold text-gray-950 dark:text-white">{userLabel}</p>
							{#if user?.email}
								<p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">{user.email}</p>
							{/if}
						</div>
						<div class="my-1 border-t border-gray-100 dark:border-gray-800"></div>
						<ActionButton variant="ghost" size="xs" full className="justify-start" on:click={() => theme.toggle()}>
							{$theme === 'dark' ? 'Use light appearance' : 'Use dark appearance'}
						</ActionButton>
						<ActionButton variant="ghostDanger" size="xs" full className="justify-start" loading={signingOut} loadingLabel="Signing out..." on:click={handleLogout}>
							Sign out
						</ActionButton>
					</div>
				{/if}
			</div>
		</div>
	</div>

	{#if mobileMenuOpen}
		<nav class="border-t border-gray-100 bg-white p-3 dark:border-gray-800 dark:bg-gray-950 lg:hidden" aria-label="Primary navigation">
			<div class="grid gap-1">
				{#each navItems as item}
					<a href={item.href} class={navItemClass(item.href)} aria-current={isActive(item.href) ? 'page' : undefined} on:click={() => (mobileMenuOpen = false)}>
						<svelte:component this={item.icon} class="h-4 w-4 shrink-0" aria-hidden="true" />
						{item.label}
					</a>
				{/each}
			</div>
		</nav>
	{/if}
</header>
