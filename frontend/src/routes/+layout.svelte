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
	import { navigating, page } from '$app/stores';
	import { beforeNavigate, goto } from '$app/navigation';
	import { updated } from '$app/state';
	import { api } from '$api';
	import { registerWebMCPTools } from '$lib/webmcp';
	import { mainContentLoading } from '$stores/main-loading';
	import type { User } from '$types';
	import faviconWhite from '../assets/new-assets/logoonly_white.png';

	let user: User | null = null;
	let checked = false;
	let unregisterWebMCP: (() => void) | null = null;

	$: isPublic = $page.url.pathname === '/' || $page.url.pathname === '/login' || $page.url.pathname.startsWith('/docs');
	$: createProjectWorkspace = $page.url.pathname === '/projects/new';
	$: showMainLoader = !checked || Boolean($navigating) || $mainContentLoading;
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
	<link rel="icon" type="image/png" href={faviconWhite} />
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

	:global(.app-workspace .page-shell > .surface),
	:global(.app-workspace .page-shell > .workspace-section) {
		border: 0;
		border-bottom: 1px solid var(--app-border) !important;
		border-radius: 0;
		background: var(--app-surface) !important;
	}

	:global(.app-workspace .panel-header) {
		border-color: var(--app-border) !important;
		background: var(--app-surface) !important;
	}

	:global(.app-workspace .workspace-section article) {
		background: var(--app-surface) !important;
	}

	/* Create Project is a staged workflow: keep internal field spacing but remove the legacy outer card. */
	:global(.create-project-workspace > .page-shell > div > form.surface) {
		border: 0;
		border-radius: 0;
		background: var(--app-surface) !important;
	}
</style>
