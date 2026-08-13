<script lang="ts">
	import { Moon, Sun } from '@lucide/svelte';
	import BrandLogo from '$components/BrandLogo.svelte';
	import GitHubMark from '$components/GitHubMark.svelte';
	import IconButton from '$components/IconButton.svelte';
	import { theme } from '$stores/theme';
</script>

<svelte:head>
	<title>Sign in · MyPaas</title>
</svelte:head>

<div class="login-shell relative flex min-h-screen min-h-[100dvh] items-center justify-center overflow-hidden bg-[var(--app-bg)]">
	<div class="pixel-gradient pointer-events-none absolute inset-0" aria-hidden="true"></div>
	<div class="pixel-grid pointer-events-none absolute inset-0" aria-hidden="true"></div>

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

	.pixel-gradient {
		inset: -8%;
		background-image:
			linear-gradient(90deg, color-mix(in srgb, var(--app-ink) 7%, transparent) 50%, transparent 50%),
			linear-gradient(color-mix(in srgb, var(--app-ink) 7%, transparent) 50%, transparent 50%),
			linear-gradient(90deg, color-mix(in srgb, var(--app-ink) 3.5%, transparent) 50%, transparent 50%),
			linear-gradient(color-mix(in srgb, var(--app-ink) 3.5%, transparent) 50%, transparent 50%);
		background-position: 0 0, 64px 64px, 16px 16px, 32px 32px;
		background-size: 128px 128px, 128px 128px, 64px 64px, 64px 64px;
		-webkit-mask-image: radial-gradient(ellipse at center, transparent 0 20%, rgb(0 0 0 / 0.12) 38%, rgb(0 0 0 / 0.72) 72%, #000 100%);
		mask-image: radial-gradient(ellipse at center, transparent 0 20%, rgb(0 0 0 / 0.12) 38%, rgb(0 0 0 / 0.72) 72%, #000 100%);
		opacity: 0.68;
		transform: rotate(-1.5deg) scale(1.04);
	}

	.pixel-grid {
		background-image:
			linear-gradient(to right, color-mix(in srgb, var(--app-border) 72%, transparent) 1px, transparent 1px),
			linear-gradient(to bottom, color-mix(in srgb, var(--app-border) 72%, transparent) 1px, transparent 1px);
		background-size: 32px 32px;
		-webkit-mask-image: radial-gradient(ellipse at center, transparent 0 26%, rgb(0 0 0 / 0.08) 48%, #000 100%);
		mask-image: radial-gradient(ellipse at center, transparent 0 26%, rgb(0 0 0 / 0.08) 48%, #000 100%);
		opacity: 0.62;
	}

	.login-panel {
		background: color-mix(in srgb, var(--app-surface) 94%, transparent);
		backdrop-filter: blur(10px);
	}

	@media (prefers-reduced-motion: reduce) {
		.pixel-gradient {
			transform: none;
		}
	}
</style>
