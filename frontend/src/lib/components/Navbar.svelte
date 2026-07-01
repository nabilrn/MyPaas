<script lang="ts">
	import { theme } from '$stores/theme';
	import type { User } from '$types';

	export let user: User | null = null;

	let menuOpen = false;
</script>

<nav class="sticky top-0 z-40 h-14 border-b border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
	<div class="mx-auto flex h-full max-w-7xl items-center justify-between px-4 sm:px-6">
		<!-- Logo -->
		<a href="/projects" class="flex items-center gap-2 font-semibold text-gray-900 dark:text-white">
			<svg class="h-6 w-6 text-brand-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14M12 5l7 7-7 7" />
			</svg>
			MyPaas
		</a>

		<!-- Center nav -->
		<div class="hidden items-center gap-1 sm:flex">
			<a href="/projects"
				class="rounded-md px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-100 hover:text-gray-900
					   dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white">
				Projects
			</a>
			<a href="/admin/users"
				class="rounded-md px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-100 hover:text-gray-900
					   dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white">
				Admin
			</a>
		</div>

		<!-- Right actions -->
		<div class="flex items-center gap-2">
			<!-- Dark mode toggle -->
			<button
				on:click={() => theme.toggle()}
				class="rounded-md p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-900
					   dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white"
				aria-label="Toggle dark mode"
			>
				{#if $theme === 'dark'}
					<svg class="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
						<path d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z" />
					</svg>
				{:else}
					<svg class="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
						<path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
					</svg>
				{/if}
			</button>

			<!-- User avatar -->
			{#if user}
				<div class="relative">
					<button
						on:click={() => (menuOpen = !menuOpen)}
						class="flex items-center gap-2 rounded-md p-1 hover:bg-gray-100 dark:hover:bg-gray-800"
					>
						<img
							src={user.avatarUrl ?? ''}
							alt={user.githubUsername ?? user.email}
							class="h-7 w-7 rounded-full"
						/>
						<span class="hidden text-sm font-medium text-gray-700 dark:text-gray-300 sm:block">
							{user.githubUsername ?? user.email}
						</span>
					</button>

					{#if menuOpen}
						<div
							class="absolute right-0 mt-1 w-48 rounded-lg border border-gray-200 bg-white py-1 shadow-lg
								   dark:border-gray-700 dark:bg-gray-800"
						>
							<div class="border-b border-gray-100 px-4 py-2 dark:border-gray-700">
								<p class="text-xs text-gray-500 dark:text-gray-400">Signed in as</p>
								<p class="text-sm font-medium text-gray-900 dark:text-white truncate">{user.email}</p>
							</div>
							<button
								on:click={async () => { const { api } = await import('$api'); await api.auth.logout(); location.href = '/login'; }}
								class="block w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-gray-50 dark:text-red-400 dark:hover:bg-gray-700">
								Sign out
							</button>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</nav>
