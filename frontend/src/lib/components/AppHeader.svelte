<script lang="ts">
	import { Bell, ChevronRight, ClipboardList, FolderKanban, LogOut, Menu, Moon, Settings, Sun, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from './ActionButton.svelte';
	import IconButton from './IconButton.svelte';
	import { api } from '$api';
	import { dismissable } from '$lib/actions/dismissable';
	import { shellContext } from '$stores/shell-context';
	import { theme } from '$stores/theme';
	import type { User } from '$types';

	export let user: User | null = null;

	const navItems = [
		{ href: '/projects', label: 'Projects', icon: FolderKanban },
		{ href: '/admin/users', label: 'Users', icon: Users },
		{ href: '/admin/audit-logs', label: 'Audit', icon: ClipboardList },
		{ href: '/admin/settings', label: 'Settings', icon: Settings }
	];

	const projectSectionLabels: Record<string, string> = {
		deployments: 'Deployments',
		logs: 'Logs',
		metrics: 'Metrics',
		env: 'Environment',
		database: 'Database',
		settings: 'Settings'
	};

	let mobileMenuOpen = false;
	let notificationsOpen = false;
	let signingOut = false;

	$: pathname = $page.url.pathname;
	$: headerContext = resolveHeaderContext(pathname, $shellContext);
	$: userLabel = user?.githubUsername ?? user?.email ?? 'Account';

	function resolveHeaderContext(currentPath: string, context: { projectId?: string; projectName?: string }) {
		if (currentPath === '/projects/new') {
			return { root: 'Projects', rootHref: '/projects', middle: null as { label: string; href: string } | null, current: 'New project' };
		}
		const projectMatch = currentPath.match(/^\/projects\/([^/]+)(?:\/([^/]+))?/);
		if (projectMatch) {
			const section = projectMatch[2] ? (projectSectionLabels[projectMatch[2]] ?? 'Project') : '';
			const projectLabel = context.projectName ?? 'Project';
			const projectHref = `/projects/${projectMatch[1]}`;
			return {
				root: 'Projects',
				rootHref: '/projects',
				middle: section ? { label: projectLabel, href: projectHref } : null,
				current: section || projectLabel
			};
		}
		if (currentPath.startsWith('/admin/users')) return { root: 'Users', rootHref: '/admin/users', middle: null, current: '' };
		if (currentPath.startsWith('/admin/audit-logs')) return { root: 'Audit', rootHref: '/admin/audit-logs', middle: null, current: '' };
		if (currentPath.startsWith('/admin/settings')) return { root: 'Settings', rootHref: '/admin/settings', middle: null, current: '' };
		return { root: 'Projects', rootHref: '/projects', middle: null, current: '' };
	}

	function isActive(href: string) {
		if (href === '/projects') return pathname === '/projects' || pathname.startsWith('/projects/');
		return pathname === href || pathname.startsWith(`${href}/`);
	}

	function navItemClass(href: string) {
		const base = 'flex min-h-10 items-center gap-3 rounded-md border px-3 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-950 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-white dark:focus-visible:ring-offset-neutral-950';
		const active = 'border-gray-200 bg-gray-100 text-gray-950 dark:border-neutral-800 dark:bg-neutral-900 dark:text-white';
		const idle = 'border-transparent text-gray-600 hover:border-gray-200 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:border-neutral-800 dark:hover:bg-neutral-900 dark:hover:text-white';
		return `${base} ${isActive(href) ? active : idle}`;
	}

	function closeNotifications() {
		notificationsOpen = false;
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

<header class="sticky top-0 z-30 border-b border-gray-200 bg-white/95 backdrop-blur dark:border-neutral-800 dark:bg-neutral-950/95">
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
			<nav class="flex min-w-0 items-center gap-1.5 text-sm" aria-label="Breadcrumb">
				<h1 class="flex min-w-0 items-center gap-1.5 text-sm font-normal">
					{#if headerContext.current}
						<a href={headerContext.rootHref} class="truncate font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">{headerContext.root}</a>
						<ChevronRight class="h-3.5 w-3.5 shrink-0 text-gray-300 dark:text-gray-700" aria-hidden="true" />
						{#if headerContext.middle}
							<a href={headerContext.middle.href} class="truncate font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">{headerContext.middle.label}</a>
							<ChevronRight class="h-3.5 w-3.5 shrink-0 text-gray-300 dark:text-gray-700" aria-hidden="true" />
						{/if}
						<span class="truncate font-semibold text-gray-950 dark:text-white" aria-current="page">{headerContext.current}</span>
					{:else}
						<span class="truncate font-semibold text-gray-950 dark:text-white" aria-current="page">{headerContext.root}</span>
					{/if}
				</h1>
			</nav>
		</div>

		<div class="relative" use:dismissable={{ enabled: notificationsOpen, onDismiss: closeNotifications }}>
			<IconButton label="Notifications" variant="ghost" on:click={() => (notificationsOpen = !notificationsOpen)}>
				<Bell class="h-4 w-4" aria-hidden="true" />
			</IconButton>
			{#if notificationsOpen}
				<div class="overlay absolute right-0 mt-2 w-[min(22rem,calc(100vw-2rem))] overflow-hidden">
					<div class="border-b border-gray-100 px-4 py-3 dark:border-neutral-800">
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
	</div>

	{#if mobileMenuOpen}
		<div class="border-t border-gray-100 bg-white p-3 dark:border-neutral-800 dark:bg-neutral-950 lg:hidden">
			<nav class="grid gap-1" aria-label="Primary navigation">
				{#each navItems as item}
					<a href={item.href} class={navItemClass(item.href)} aria-current={isActive(item.href) ? 'page' : undefined} on:click={() => (mobileMenuOpen = false)}>
						<svelte:component this={item.icon} class="h-4 w-4 shrink-0" aria-hidden="true" />
						{item.label}
					</a>
				{/each}
			</nav>

			<div class="mt-3 border-t border-gray-100 pt-3 dark:border-neutral-800">
				<div class="px-3 pb-2">
					<p class="truncate text-sm font-medium text-gray-950 dark:text-white">{userLabel}</p>
					{#if user?.email}
						<p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">{user.email}</p>
					{/if}
				</div>
				<div class="grid gap-1">
					<ActionButton variant="ghost" size="sm" full className="justify-start" on:click={() => theme.toggle()}>
						<svelte:fragment slot="icon">
							{#if $theme === 'dark'}
								<Sun class="h-4 w-4" />
							{:else}
								<Moon class="h-4 w-4" />
							{/if}
						</svelte:fragment>
						{$theme === 'dark' ? 'Light appearance' : 'Dark appearance'}
					</ActionButton>
					<ActionButton variant="ghostDanger" size="sm" full className="justify-start" loading={signingOut} loadingLabel="Signing out..." on:click={handleLogout}>
						<LogOut slot="icon" class="h-4 w-4" />
						Sign out
					</ActionButton>
				</div>
			</div>
		</div>
	{/if}
</header>
