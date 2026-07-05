<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { theme } from '$stores/theme';
	import type { User } from '$types';

	export let user: User | null = null;

	let menuOpen = false;
	let signingOut = false;

	const navItems = [
		{ href: '/projects', label: 'Projects' },
		{ href: '/admin/users', label: 'Users' },
		{ href: '/admin/audit-logs', label: 'Audit' }
	];

	$: pathname = $page.url.pathname;

	function isActive(href: string) {
		return href === '/projects' ? pathname === '/projects' || pathname.startsWith('/projects/') : pathname.startsWith(href);
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

<nav class="sticky top-0 z-40 border-b border-gray-200/80 bg-white/90 backdrop-blur dark:border-gray-800/80 dark:bg-gray-950/90">
	<div class="mx-auto flex h-14 max-w-7xl items-center justify-between px-4 sm:px-6">
		<a href="/projects" class="flex items-center gap-2.5 font-semibold text-gray-950 dark:text-white">
			<span class="flex h-7 w-7 items-center justify-center rounded-md bg-gray-950 text-white dark:bg-white dark:text-gray-950">
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.25">
					<path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14M12 5l7 7-7 7" />
				</svg>
			</span>
			<span class="text-sm tracking-tight">MyPaas</span>
		</a>

		<div class="hidden items-center rounded-md bg-gray-100 p-0.5 dark:bg-gray-900 sm:flex">
			{#each navItems as item}
				<a
					href={item.href}
					aria-current={isActive(item.href) ? 'page' : undefined}
					class="rounded px-3 py-1.5 text-sm font-medium transition-colors
						{isActive(item.href)
							? 'bg-white text-gray-950 shadow-sm dark:bg-gray-800 dark:text-white'
							: 'text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white'}"
				>
					{item.label}
				</a>
			{/each}
		</div>

		<div class="flex items-center gap-2">
			<button
				on:click={() => theme.toggle()}
				class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-gray-200 text-gray-500 hover:bg-gray-50 hover:text-gray-950 dark:border-gray-800 dark:text-gray-400 dark:hover:bg-gray-900 dark:hover:text-white"
				aria-label="Toggle dark mode"
			>
				{#if $theme === 'dark'}
					<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
						<path d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z" />
					</svg>
				{:else}
					<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
						<path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
					</svg>
				{/if}
			</button>

			{#if user}
				<div class="relative">
					<button
						on:click={() => (menuOpen = !menuOpen)}
						class="flex h-9 items-center gap-2 rounded-md border border-gray-200 bg-white px-2 text-sm hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950 dark:hover:bg-gray-900"
					>
						{#if user.avatarUrl}
							<img src={user.avatarUrl} alt={user.githubUsername ?? user.email} class="h-6 w-6 rounded-full" />
						{:else}
							<span class="flex h-6 w-6 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-500 dark:bg-gray-800 dark:text-gray-300">
								{(user.email || '?').slice(0, 1).toUpperCase()}
							</span>
						{/if}
						<span class="hidden max-w-32 truncate text-gray-700 dark:text-gray-300 sm:block">
							{user.githubUsername ?? user.email}
						</span>
					</button>

					{#if menuOpen}
						<div class="absolute right-0 mt-2 w-56 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-lg shadow-gray-950/10 dark:border-gray-800 dark:bg-gray-900">
							<div class="border-b border-gray-100 px-4 py-3 dark:border-gray-800">
								<p class="text-xs text-gray-500 dark:text-gray-400">Signed in as</p>
								<p class="mt-0.5 truncate text-sm font-medium text-gray-950 dark:text-white">{user.email}</p>
							</div>
							<ActionButton
								variant="ghostDanger"
								size="xs"
								full
								on:click={handleLogout}
								loading={signingOut}
								loadingLabel="Signing out..."
								className="justify-start rounded-none px-4 py-2.5 text-left"
							>
								Sign out
							</ActionButton>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</nav>
