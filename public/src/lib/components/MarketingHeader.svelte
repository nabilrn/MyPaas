<script lang="ts">
	import { BookOpen, GitBranch, LayoutDashboard, Menu, Moon, Sun, X } from '@lucide/svelte';
	import { theme } from '$stores/theme';
	import logoGreen from '../../assets/mypaas-horizontal-transparent-green.png';
	import logoWhite from '../../assets/mypaas-horizontal-transparent-white.png';

	export let active: 'home' | 'docs' = 'home';
	let open = false;
</script>

<header class="sticky top-0 z-40 border-b border-gray-200/90 bg-white/95 backdrop-blur dark:border-gray-800 dark:bg-gray-950/95">
	<div class="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
		<a href="/" class="app-focus rounded-md">
			<span class="sr-only">MyPaas home</span>
			<img src={logoGreen} alt="" class="h-9 w-[138px] object-contain object-left dark:hidden" />
			<img src={logoWhite} alt="" class="hidden h-9 w-[138px] object-contain object-left dark:block" />
		</a>

		<nav class="hidden items-center gap-1 md:flex" aria-label="Primary navigation">
			<a href="/" aria-current={active === 'home' ? 'page' : undefined} class:marketing-active-link={active === 'home'} class="marketing-nav-link">Product</a>
			<a href="/#integrations" class="marketing-nav-link">Integrations</a>
			<a href="/docs" aria-current={active === 'docs' ? 'page' : undefined} class:marketing-active-link={active === 'docs'} class="marketing-nav-link">Docs</a>
		</nav>

		<div class="hidden items-center gap-2 md:flex">
			<a href="https://github.com/nabilrn/mypaas" target="_blank" rel="noreferrer" class="marketing-icon-link" aria-label="MyPaas on GitHub"><GitBranch size={17} aria-hidden="true" /></a>
			<button class="marketing-icon-link" type="button" aria-label="Toggle color theme" on:click={() => theme.toggle()}>
				{#if $theme === 'dark'}<Sun size={17} aria-hidden="true" />{:else}<Moon size={17} aria-hidden="true" />{/if}
			</button>
			<a href="/login" class="inline-flex h-9 items-center gap-2 rounded-md bg-brand-700 px-3.5 text-sm font-semibold text-white transition-colors hover:bg-brand-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 dark:bg-brand-500 dark:text-gray-950 dark:hover:bg-brand-100">
				<LayoutDashboard size={16} aria-hidden="true" /> Open dashboard
			</a>
		</div>

		<button type="button" class="marketing-icon-link md:hidden" aria-label="Toggle navigation" aria-expanded={open} on:click={() => (open = !open)}>
			{#if open}<X size={19} aria-hidden="true" />{:else}<Menu size={19} aria-hidden="true" />{/if}
		</button>
	</div>

	{#if open}
		<nav class="border-t border-gray-200 p-3 dark:border-gray-800 md:hidden" aria-label="Mobile navigation">
			<div class="grid gap-1">
				<a href="/" class="marketing-mobile-link" on:click={() => (open = false)}>Product</a>
				<a href="/#integrations" class="marketing-mobile-link" on:click={() => (open = false)}>Integrations</a>
				<a href="/docs" class="marketing-mobile-link" on:click={() => (open = false)}><BookOpen size={16} aria-hidden="true" /> Docs</a>
				<a href="/login" class="marketing-mobile-link" on:click={() => (open = false)}><LayoutDashboard size={16} aria-hidden="true" /> Open dashboard</a>
			</div>
		</nav>
	{/if}
</header>
