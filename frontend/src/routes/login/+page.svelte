<script lang="ts">
	// Login is handled by GitHub OAuth — this page just shows the entry point.
</script>

<svelte:head>
	<title>Sign in · MyPaas</title>
</svelte:head>

<div class="flex min-h-screen flex-col items-center justify-center bg-gray-50 dark:bg-gray-950">
	<div class="w-full max-w-sm space-y-8 px-4">
		<!-- Logo -->
		<div class="text-center">
			<div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-gray-950 text-white dark:bg-white dark:text-gray-950">
				<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14M12 5l7 7-7 7" />
				</svg>
			</div>
			<h1 class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">MyPaas</h1>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
				Self-hosted deployment platform
			</p>
		</div>

		<!-- Card -->
		<div class="surface p-8">
			<h2 class="mb-6 text-center text-lg font-semibold text-gray-950 dark:text-white">
				Sign in to continue
			</h2>

			<a
				href="/api/auth/github/login"
				class="flex w-full items-center justify-center gap-3 rounded-md border border-gray-300 bg-white
					   px-4 py-2.5 text-sm font-medium text-gray-800 shadow-sm transition-colors
					   hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-950 focus:ring-offset-2
					   dark:border-gray-700 dark:bg-gray-950 dark:text-white dark:hover:bg-gray-900 dark:focus:ring-white dark:focus:ring-offset-gray-950"
			>
				<svg class="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
				</svg>
				Continue with GitHub
			</a>

			<p class="mt-4 text-center text-xs text-gray-400 dark:text-gray-500">
				Access is restricted to whitelisted accounts only.
			</p>
		</div>
	</div>
</div>
