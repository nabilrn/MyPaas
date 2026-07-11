<script lang="ts">
	import { onMount } from 'svelte';
	import '../app.css';
	import Navbar from '$components/Navbar.svelte';
	import Toast  from '$components/Toast.svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { api } from '$api';
	import { sidebarCollapsed } from '$stores/sidebar';
	import { theme } from '$stores/theme';
	import type { User } from '$types';
	import faviconGreen from '../assets/mypaas-icon-transparent-green.png';
	import faviconWhite from '../assets/mypaas-icon-transparent-white.png';

	let user: User | null = null;
	let checked = false;

	$: isPublic = $page.url.pathname === '/' || $page.url.pathname === '/login' || $page.url.pathname.startsWith('/docs');

	onMount(async () => {
		if (isPublic) {
			checked = true;
			return;
		}
		try {
			user = await api.auth.me();
		} catch {
			await goto('/login');
		} finally {
			checked = true;
		}
	});
</script>

<svelte:head>
	<link rel="icon" type="image/png" href={$theme === 'dark' ? faviconWhite : faviconGreen} />
</svelte:head>

{#if checked || isPublic}
	{#if !isPublic && user}
		<div class="min-h-screen transition-[padding] duration-200 {$sidebarCollapsed ? 'lg:pl-16' : 'lg:pl-64'}">
			<Navbar {user} />
			<main class="min-h-screen">
				<slot />
			</main>
		</div>
	{:else}
		<main class="min-h-screen">
			<slot />
		</main>
	{/if}
{/if}

<Toast />
