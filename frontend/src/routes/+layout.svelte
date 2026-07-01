<script lang="ts">
	import { onMount } from 'svelte';
	import '../app.css';
	import Navbar from '$components/Navbar.svelte';
	import Toast  from '$components/Toast.svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { api } from '$api';
	import type { User } from '$types';

	let user: User | null = null;
	let checked = false;

	// Hide navbar on the login page
	$: isLogin = $page.url.pathname === '/login';

	onMount(async () => {
		if (isLogin) {
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

{#if !isLogin && user}
	<Navbar {user} />
{/if}

<main class="min-h-screen bg-gray-50 dark:bg-gray-950">
	{#if checked || isLogin}
		<slot />
	{/if}
</main>

<Toast />
