<script lang="ts">
	import { onMount } from 'svelte';
	import '@fontsource-variable/inter';
	import '@fontsource/ibm-plex-mono/400.css';
	import '@fontsource/ibm-plex-mono/500.css';
	import '../app.css';
	import AppHeader from '$components/AppHeader.svelte';
	import MainContentLoader from '$components/MainContentLoader.svelte';
	import Navbar from '$components/Navbar.svelte';
	import Toast from '$components/Toast.svelte';
	import { page } from '$app/stores';
	import { beforeNavigate, goto } from '$app/navigation';
	import { updated } from '$app/state';
	import { api } from '$api';
	import { registerWebMCPTools } from '$lib/webmcp';
	import { mainContentLoading } from '$stores/main-loading';
	import type { User } from '$types';
	import favicon from '../assets/brand/mypaas-icon.svg';

	let user: User | null = null;
	let checked = false;
	let unregisterWebMCP: (() => void) | null = null;

	$: isPublic = $page.url.pathname === '/' || $page.url.pathname === '/login' || $page.url.pathname.startsWith('/docs');
	$: createProjectWorkspace = $page.url.pathname === '/projects/new';
	$: showMainLoader = !checked || $mainContentLoading;
	$: if (isPublic && unregisterWebMCP) {
		unregisterWebMCP();
		unregisterWebMCP = null;
	}

	beforeNavigate(({ willUnload, to }) => {
		if (updated.current && !willUnload && to?.url) {
			location.href = to.url.href;
		}
	});

	onMount(() => {
		void bootstrap();
		return () => unregisterWebMCP?.();
	});

	async function bootstrap() {
		if (isPublic) {
			checked = true;
			return;
		}
		try {
			user = await api.auth.me();
			unregisterWebMCP?.();
			unregisterWebMCP = registerWebMCPTools(user);
		} catch {
			unregisterWebMCP?.();
			unregisterWebMCP = null;
			await goto('/login');
		} finally {
			checked = true;
		}
	}
</script>

<svelte:head>
	<link rel="icon" type="image/svg+xml" href={favicon} />
</svelte:head>

{#if isPublic}
	<main class="min-h-screen">
		<slot />
	</main>
{:else}
	<div class="app-shell min-h-screen lg:pl-14">
		<Navbar {user} authPending={!checked} />
		<AppHeader {user} />
		<main class="app-workspace relative min-h-[calc(100vh-3.5rem)]" aria-busy={showMainLoader}>
			{#if checked && user}
				<div class:create-project-workspace={createProjectWorkspace} class:pointer-events-none={showMainLoader}>
					<slot />
				</div>
			{/if}
			{#if showMainLoader}<MainContentLoader label={checked ? 'Loading' : 'Loading account'} />{/if}
		</main>
	</div>
{/if}

<Toast />

<style>
	:global(.app-shell) {
		--technical-surface-bg: #171717;
		--technical-surface-border: #374151;
		--technical-surface-text: #e5e7eb;
		--technical-surface-muted: #9ca3af;
		--control-height: 2.25rem;
		--control-radius: 0.375rem;
		--control-font-size: 0.875rem;
	}

	:global(.dark .app-shell) {
		--technical-surface-bg: #0a0a0a;
		--technical-surface-border: #262626;
		--technical-surface-text: #e5e7eb;
		--technical-surface-muted: #9ca3af;
	}

	/* Every ordinary single-line dashboard control shares one geometry contract. */
	:global(.app-shell :is(input, select).field),
	:global(.app-shell [data-action-button]),
	:global(.app-shell [data-action-link]),
	:global(.app-shell .control-height) {
		height: var(--control-height) !important;
		min-height: var(--control-height) !important;
		border-radius: var(--control-radius) !important;
		font-size: var(--control-font-size) !important;
	}

	:global(.app-shell [data-icon-button]),
	:global(.app-shell .control-square) {
		width: var(--control-height) !important;
		height: var(--control-height) !important;
		min-width: var(--control-height) !important;
		min-height: var(--control-height) !important;
		border-radius: var(--control-radius) !important;
	}

	@media (any-pointer: coarse) {
		:global(.app-shell) {
			--control-height: 2.75rem;
		}
	}

	:global(.app-shell),
	:global(.app-shell > aside),
	:global(.app-shell > header),
	:global(.app-workspace) {
		background: var(--app-surface) !important;
	}

	:global(.app-shell > aside),
	:global(.app-shell > header) {
		border-color: var(--app-border) !important;
	}

	:global(.app-workspace .page-shell) {
		width: 100%;
		max-width: none;
		margin: 0;
		padding: 0 !important;
		background: var(--app-surface);
	}

	/* Authenticated sections share one fill. Hierarchy belongs to strokes, not tonal blocks. */
	:global(.app-workspace .workspace-section),
	:global(.app-workspace .surface),
	:global(.app-workspace .surface-muted),
	:global(.app-workspace .soft-panel),
	:global(.app-workspace .control-panel) {
		background: var(--app-surface) !important;
	}

	:global(.app-workspace .page-shell > .surface),
	:global(.app-workspace .page-shell > .workspace-section) {
		border: 0;
		border-bottom: 1px solid var(--app-border) !important;
		border-radius: 0;
		background: var(--app-surface) !important;
	}

	:global(.app-workspace .panel-header),
	:global(.app-workspace .table-toolbar),
	:global(.app-workspace .data-table),
	:global(.app-workspace .data-table thead th),
	:global(.app-workspace .data-table tbody tr),
	:global(.app-workspace .chip),
	:global(.app-workspace .alert-neutral) {
		background: var(--app-surface) !important;
	}

	:global(.app-workspace .panel-header),
	:global(.app-workspace .table-toolbar) {
		border-color: var(--app-border) !important;
	}

	:global(.app-workspace .data-table tbody tr:hover),
	:global(.app-workspace .data-table tbody tr[aria-selected='true']),
	:global(.app-workspace .data-table tbody tr[data-selected='true']) {
		background: var(--app-surface) !important;
	}

	:global(.app-workspace .data-table tbody tr[aria-selected='true']),
	:global(.app-workspace .data-table tbody tr[data-selected='true']) {
		box-shadow: inset 2px 0 0 var(--app-border-strong);
	}

	/* Legacy neutral fills are normalized only inside operational sections/full-canvas DB chrome. */
	:global(.app-workspace .workspace-section [class~='bg-white']),
	:global(.app-workspace .workspace-section [class~='bg-gray-50']),
	:global(.app-workspace .workspace-section [class~='bg-gray-50/40']),
	:global(.app-workspace .workspace-section [class~='bg-gray-50/45']),
	:global(.app-workspace .workspace-section [class~='bg-gray-50/50']),
	:global(.app-workspace .workspace-section [class~='bg-gray-50/60']),
	:global(.app-workspace .workspace-section [class~='bg-gray-50/70']),
	:global(.app-workspace .workspace-section [class~='bg-gray-50/80']),
	:global(.app-workspace .workspace-section [class~='bg-gray-100']),
	:global(.app-workspace .workspace-section [class~='bg-gray-100/70']),
	:global(.app-workspace .workspace-section [class~='dark:bg-gray-900']),
	:global(.app-workspace .workspace-section [class~='dark:bg-neutral-900']),
	:global(.app-workspace .workspace-section [class~='dark:bg-neutral-900/40']),
	:global(.app-workspace .workspace-section [class~='dark:bg-neutral-900/45']),
	:global(.app-workspace .workspace-section [class~='dark:bg-neutral-900/50']),
	:global(.app-workspace .workspace-section [class~='dark:bg-neutral-900/60']),
	:global(.app-workspace .workspace-section [class~='dark:bg-neutral-900/70']),
	:global(.app-workspace .workspace-section [class~='dark:bg-neutral-950']),
	:global(.app-workspace .workspace-section [class~='dark:bg-neutral-950/40']),
	:global(.app-workspace .workspace-section [class~='hover:bg-gray-50']),
	:global(.app-workspace .workspace-section [class~='hover:bg-gray-50/80']),
	:global(.app-workspace .workspace-section [class~='hover:bg-gray-100']),
	:global(.app-workspace .workspace-section [class~='dark:hover:bg-neutral-800']),
	:global(.app-workspace .workspace-section [class~='dark:hover:bg-neutral-900']),
	:global(.app-workspace .workspace-section [class~='dark:hover:bg-neutral-900/70']),
	:global(.app-workspace .database-design-shell [class~='bg-white']),
	:global(.app-workspace .database-design-shell [class~='bg-gray-50']),
	:global(.app-workspace .database-design-shell [class~='bg-gray-100']),
	:global(.app-workspace .database-design-shell [class~='dark:bg-neutral-900']),
	:global(.app-workspace .database-design-shell [class~='dark:bg-neutral-950']) {
		background-color: var(--app-surface) !important;
	}

	/* Divider grids retain only the 1px divider; every cell uses the workspace fill. */
	:global(.app-workspace .workspace-section .grid.gap-px) {
		background-color: var(--workspace-divider) !important;
	}

	:global(.app-workspace .workspace-section .grid.gap-px > *) {
		background-color: var(--app-surface) !important;
	}

	:global(.app-workspace .workspace-section [aria-current='true']),
	:global(.app-workspace .workspace-section [aria-current='page']) {
		box-shadow: inset 2px 0 0 var(--app-border-strong);
	}

	:global(.app-workspace .workspace-section button[aria-pressed='true']) {
		border-color: var(--app-border-strong) !important;
	}

	/* Full-canvas database chrome follows the same workspace fill. */
	:global(.app-workspace .database-design-shell),
	:global(.app-workspace .database-design-shell > header) {
		background: var(--app-surface) !important;
		border-color: var(--app-border) !important;
	}

	/* Host Shell is the canonical palette for terminal/log/build/preformatted output blocks. */
	:global(.app-workspace .console-surface),
	:global(.app-workspace pre) {
		background: var(--technical-surface-bg) !important;
		border-color: var(--technical-surface-border) !important;
		color: var(--technical-surface-text) !important;
	}

	:global(.app-workspace pre:not(.console-surface)) {
		border-radius: 0.375rem;
	}

	:global(.app-workspace .console-surface pre) {
		border: 0 !important;
		border-radius: 0 !important;
	}

	:global(.app-workspace .console-surface .text-gray-500),
	:global(.app-workspace pre .text-gray-500) {
		color: var(--technical-surface-muted) !important;
	}

	/* Create Project is a staged workflow: keep internal field spacing but remove the legacy outer card. */
	:global(.create-project-workspace > .page-shell > div > form.surface) {
		border: 0;
		border-radius: 0;
		background: var(--app-surface) !important;
	}
</style>
