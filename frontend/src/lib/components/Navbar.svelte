<script lang="ts">
	import { ArrowRightLeft, Bot, Boxes, ClipboardList, Database, Layers3, Network, Settings, Terminal, Users } from '@lucide/svelte';
	import { page } from '$app/stores';
	import BrandLogo from '$components/BrandLogo.svelte';
	import type { User } from '$types';

	export let user: User | null = null;
	export let authPending = false;

	const workspaceItems = [
		{ href: '/projects', label: 'Projects', icon: Layers3, ownerOnly: false },
		{ href: '/containers', label: 'Containers', icon: Boxes, ownerOnly: false },
		{ href: '/ports', label: 'Ports', icon: Network, ownerOnly: true },
		{ href: '/shell', label: 'Shell', icon: Terminal, ownerOnly: true }
	];

	const administrationItems = [
		{ href: '/admin/users', label: 'Users', icon: Users, ownerOnly: true },
		{ href: '/admin/audit-logs', label: 'Audit', icon: ClipboardList, ownerOnly: true },
		{ href: '/admin/mcp', label: 'MCP', icon: Bot, ownerOnly: true },
		{ href: '/admin/backup', label: 'Backup', icon: Database, ownerOnly: true },
		{ href: '/admin/migration', label: 'Migration', icon: ArrowRightLeft, ownerOnly: true },
		{ href: '/admin/settings', label: 'Settings', icon: Settings, ownerOnly: true }
	];

	let expanded = false;
	let sidebar: HTMLElement | null = null;

	$: pathname = $page.url.pathname;
	$: visibleWorkspaceItems = workspaceItems.filter((item) => authPending || !item.ownerOnly || user?.role === 'owner');
	$: visibleAdministrationItems = administrationItems.filter((item) => authPending || !item.ownerOnly || user?.role === 'owner');

	function isActive(href: string, currentPath: string) {
		if (href === '/projects') return currentPath === '/projects' || currentPath.startsWith('/projects/');
		return currentPath === href || currentPath.startsWith(`${href}/`);
	}

	function navItemClass(href: string, isExpanded: boolean, currentPath: string) {
		const layout = isExpanded ? 'justify-start gap-2.5 px-3' : 'justify-center px-0';
		const base = `group relative flex min-h-9 w-full items-center rounded-md text-left text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-950 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-white dark:focus-visible:ring-offset-neutral-950 ${layout}`;
		const active = 'bg-gray-100 text-gray-950 dark:bg-neutral-900 dark:text-white';
		const idle = 'text-gray-600 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-neutral-900 dark:hover:text-white';
		return `${base} ${isActive(href, currentPath) ? active : idle}`;
	}

	function handleFocusOut(event: FocusEvent) {
		const next = event.relatedTarget;
		if (!sidebar || !(next instanceof Node) || !sidebar.contains(next)) expanded = false;
	}

	function authorizationPending(ownerOnly: boolean) {
		return authPending && ownerOnly && user === null;
	}

	function chooseNavigation(event: MouseEvent, ownerOnly: boolean) {
		if (authorizationPending(ownerOnly)) {
			event.preventDefault();
			return;
		}
		expanded = false;
	}
</script>

<aside
	bind:this={sidebar}
	class={`fixed inset-y-0 left-0 z-40 hidden flex-col overflow-hidden border-r border-gray-100/80 bg-white transition-[width,box-shadow] duration-150 ease-out dark:border-neutral-900 dark:bg-neutral-950 lg:flex ${expanded ? 'w-60 shadow-[10px_0_24px_rgb(0_0_0/0.07)] dark:shadow-[10px_0_24px_rgb(0_0_0/0.24)]' : 'w-14 shadow-none'}`}
	on:mouseenter={() => (expanded = true)}
	on:mouseleave={() => (expanded = false)}
	on:focusin={() => (expanded = true)}
	on:focusout={handleFocusOut}
>
	<div class={`flex h-14 shrink-0 items-center border-b border-gray-100/80 dark:border-neutral-900 ${expanded ? 'px-4' : 'justify-center px-2'}`}>
		<a
			href="/projects"
			class="app-focus flex h-9 min-w-0 items-center rounded-md"
			aria-label="MyPaaS projects"
			on:click={(event) => chooseNavigation(event, false)}
		>
			<BrandLogo compact={!expanded} />
		</a>
	</div>

	<nav class={`flex-1 overflow-y-auto py-3 ${expanded ? 'px-3' : 'px-2'}`} aria-label="Primary navigation">
		{#if expanded}<p class="px-3 pb-1.5 text-[13px] font-medium text-gray-400 dark:text-gray-500">Workspace</p>{/if}
		<div class="space-y-1">
			{#each visibleWorkspaceItems as item}
				<a
					href={item.href}
					aria-current={isActive(item.href, pathname) ? 'page' : undefined}
					aria-disabled={authorizationPending(item.ownerOnly) ? 'true' : undefined}
					tabindex={authorizationPending(item.ownerOnly) ? -1 : undefined}
					class={navItemClass(item.href, expanded, pathname)}
					title={expanded ? undefined : item.label}
					on:click={(event) => chooseNavigation(event, item.ownerOnly)}
				>
					<svelte:component this={item.icon} class="h-[18px] w-[18px] shrink-0" aria-hidden="true" />
					{#if expanded}<span class="min-w-0 truncate whitespace-nowrap">{item.label}</span>{:else}<span class="sr-only">{item.label}</span>{/if}
				</a>
			{/each}
		</div>

		{#if visibleAdministrationItems.length > 0}
			<div class={`my-3 border-t border-gray-100 dark:border-neutral-900 ${expanded ? 'mx-1' : 'mx-1.5'}`}></div>
			{#if expanded}<p class="px-3 pb-1.5 text-[13px] font-medium text-gray-400 dark:text-gray-500">Administration</p>{/if}
			<div class="space-y-1">
				{#each visibleAdministrationItems as item}
					<a
						href={item.href}
						aria-current={isActive(item.href, pathname) ? 'page' : undefined}
						aria-disabled={authorizationPending(item.ownerOnly) ? 'true' : undefined}
						tabindex={authorizationPending(item.ownerOnly) ? -1 : undefined}
						class={navItemClass(item.href, expanded, pathname)}
						title={expanded ? undefined : item.label}
						on:click={(event) => chooseNavigation(event, item.ownerOnly)}
					>
						<svelte:component this={item.icon} class="h-[18px] w-[18px] shrink-0" aria-hidden="true" />
						{#if expanded}<span class="min-w-0 truncate whitespace-nowrap">{item.label}</span>{:else}<span class="sr-only">{item.label}</span>{/if}
					</a>
				{/each}
			</div>
		{/if}
	</nav>
</aside>
