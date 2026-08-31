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
	import { goto } from '$app/navigation';
	import { api } from '$api';
	import { registerWebMCPTools } from '$lib/webmcp';
	import { mainContentLoading } from '$stores/main-loading';
	import type { User } from '$types';
	import faviconWhite from '../assets/new-assets/logoonly_white.png';

	let user: User | null = null;
	let checked = false;
	let unregisterWebMCP: (() => void) | null = null;

	$: isPublic = $page.url.pathname === '/' || $page.url.pathname === '/login' || $page.url.pathname.startsWith('/docs');
	$: showMainLoader = Boolean($navigating) || $mainContentLoading;
	$: if (isPublic && unregisterWebMCP) {
		unregisterWebMCP();
		unregisterWebMCP = null;
	}

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

{#if checked || isPublic}
	{#if !isPublic && user}
		<div class="min-h-screen lg:pl-14">
			<Navbar {user} />
			<AppHeader {user} />
			<main class="relative min-h-[calc(100vh-3.5rem)]" aria-busy={showMainLoader}>
				<div
					class:invisible={showMainLoader}
					class:pointer-events-none={showMainLoader}
					class:absolute={showMainLoader}
					class:inset-0={showMainLoader}
					class:overflow-hidden={showMainLoader}
					aria-hidden={showMainLoader}
				>
					<slot />
				</div>
				{#if showMainLoader}<MainContentLoader />{/if}
			</main>
		</div>
	{:else}
		<main class="min-h-screen">
			<slot />
		</main>
	{/if}
{/if}

<Toast />
