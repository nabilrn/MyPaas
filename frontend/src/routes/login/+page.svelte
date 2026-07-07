<script lang="ts">
	import logoGreen from '../../assets/mypaas-horizontal-transparent-green.png';
	import logoWhite from '../../assets/mypaas-horizontal-transparent-white.png';
	import circuitBgLight from '../../assets/mypaas-circuit-background.svg';
	import circuitBgDark from '../../assets/mypaas-circuit-background-dark.svg';
	import { theme } from '$stores/theme';
</script>

<svelte:head>
	<title>Sign in · MyPaas</title>
</svelte:head>

<div class="login-page">
	<!-- Circuit background: light (green) / dark (white) -->
	<img
		src={circuitBgLight}
		alt=""
		aria-hidden="true"
		class="circuit-bg pointer-events-none dark:hidden"
	/>
	<img
		src={circuitBgDark}
		alt=""
		aria-hidden="true"
		class="circuit-bg pointer-events-none hidden dark:block"
	/>

	<!-- Theme toggle — top-right corner -->
	<button
		type="button"
		class="theme-toggle"
		aria-label="Toggle dark mode"
		on:click={() => theme.toggle()}
	>
		{#if $theme === 'dark'}
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<circle cx="12" cy="12" r="4" />
				<path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
			</svg>
		{:else}
			<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
				<path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
			</svg>
		{/if}
	</button>

	<!-- Centered login content -->
	<main class="login-content">
		<div class="login-inner">
			<div class="mb-6 flex flex-col items-center text-center">
				<div class="flex h-14 w-[200px] items-center justify-center">
					<img src={logoGreen} alt="MyPaas" class="h-14 w-[200px] object-contain dark:hidden" />
					<img src={logoWhite} alt="MyPaas" class="hidden h-14 w-[200px] object-contain dark:block" />
				</div>
				<p class="mt-2 text-sm" style="color: var(--app-muted);">
					Self-hosted Git-based deployments.
				</p>
				<h1 class="sr-only">Sign in to MyPaas</h1>
			</div>

			<div class="login-card">
				<a
					href="/api/auth/github/login"
					id="login-github-btn"
					class="github-btn"
				>
					<svg class="h-5 w-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
					</svg>
					Continue with GitHub
				</a>
			</div>
		</div>
	</main>
</div>

<style>
	.login-page {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
		min-height: 100dvh;
		overflow: hidden;
		background: var(--app-bg);
	}

	/* Circuit background — covers viewport */
	.circuit-bg {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
		opacity: 0.5;
	}

	/* Theme toggle button */
	.theme-toggle {
		position: absolute;
		top: 1rem;
		right: 1rem;
		z-index: 2;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border: 1px solid var(--app-border);
		border-radius: 0.375rem;
		background: var(--app-surface);
		color: var(--app-accent);
		cursor: pointer;
		transition: border-color 0.15s, color 0.15s;
	}

	.theme-toggle:hover {
		border-color: var(--app-border-strong);
		color: var(--app-accent-strong);
	}

	.theme-toggle:focus-visible {
		outline: none;
		border-color: var(--app-accent);
		box-shadow: 0 0 0 1px var(--app-accent),
			0 0 0 4px color-mix(in oklch, var(--app-accent-soft) 76%, transparent);
	}

	/* Login content — always on top */
	.login-content {
		position: relative;
		z-index: 1;
		width: 100%;
		max-width: 22rem;
		padding: 1.25rem;
	}

	.login-inner {
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.login-card {
		width: 100%;
		border: 1px solid var(--app-border);
		border-radius: 0.5rem;
		background: var(--app-surface);
		padding: 0.75rem;
		box-shadow: 0 1px 2px rgb(15 23 42 / 0.03);
	}

	.github-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		width: 100%;
		min-height: 2.75rem;
		padding: 0.625rem 1rem;
		border: 1px solid var(--app-border);
		border-radius: 0.375rem;
		background: var(--app-surface);
		color: var(--app-ink);
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
		transition: border-color 0.15s, background 0.15s;
	}

	.github-btn:hover {
		border-color: var(--app-border-strong);
		background: var(--app-surface-muted);
	}

	.github-btn:focus-visible {
		outline: none;
		border-color: var(--app-accent);
		box-shadow: 0 0 0 1px var(--app-accent),
			0 0 0 4px color-mix(in oklch, var(--app-accent-soft) 76%, transparent);
	}
</style>
