<script lang="ts">
	import { Moon, Sun } from '@lucide/svelte';
	import BrandLogo from '$components/BrandLogo.svelte';
	import GitHubMark from '$components/GitHubMark.svelte';
	import IconButton from '$components/IconButton.svelte';
	import loginBackground from '../../assets/mypaas-login-pixel-background.webp';
	import { theme } from '$stores/theme';
</script>

<svelte:head>
	<title>Sign in · MyPaas</title>
</svelte:head>

<div class="login-shell relative flex min-h-screen min-h-[100dvh] items-center justify-center overflow-hidden bg-[var(--app-bg)]">
	<img src={loginBackground} alt="" aria-hidden="true" class="login-background pointer-events-none absolute inset-0 h-full w-full object-cover" />
	<div class="login-vignette pointer-events-none absolute inset-0" aria-hidden="true"></div>

	<div class="absolute right-4 top-4 z-10">
		<IconButton label="Toggle appearance" variant="secondary" on:click={() => theme.toggle()}>
			{#if $theme === 'dark'}
				<Sun class="h-4 w-4" aria-hidden="true" />
			{:else}
				<Moon class="h-4 w-4" aria-hidden="true" />
			{/if}
		</IconButton>
	</div>

	<main class="relative z-[1] w-full max-w-[24rem] px-5 py-10">
		<div class="mb-7 flex flex-col items-center text-center">
			<div class="flex h-20 w-[250px] items-center justify-center sm:w-[270px]">
				<BrandLogo imageClass="h-20 w-full object-contain object-center" />
			</div>
			<p class="mt-2 text-[0.9375rem] text-gray-500 dark:text-gray-400">Self-hosted Git-based deployments.</p>
			<h1 class="sr-only">Sign in to MyPaas</h1>
		</div>

		<div class="surface login-panel p-3">
			<a
				href="/api/auth/github/login"
				id="login-github-btn"
				class="app-focus flex min-h-11 w-full items-center justify-center gap-3 rounded-md border border-gray-300 bg-gray-950 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-black dark:border-white dark:bg-white dark:text-gray-950 dark:hover:bg-gray-200"
			>
				<GitHubMark className="h-5 w-5" />
				Continue with GitHub
			</a>
		</div>
	</main>
</div>

<style>
	.login-shell {
		isolation: isolate;
	}

	.login-background {
		opacity: 0.94;
		filter: grayscale(1) contrast(0.98);
	}

	.login-vignette {
		background: radial-gradient(circle at center, color-mix(in srgb, var(--app-bg) 58%, transparent) 0 14%, transparent 58%);
	}

	.login-panel {
		background: color-mix(in srgb, var(--app-surface) 95%, transparent);
		backdrop-filter: blur(12px);
	}

	:global(.dark) .login-background {
		opacity: 0.72;
		filter: grayscale(1) invert(1) brightness(0.24) contrast(1.08);
	}

	:global(.dark) .login-vignette {
		background: radial-gradient(circle at center, color-mix(in srgb, var(--app-bg) 66%, transparent) 0 16%, transparent 62%);
	}
</style>
