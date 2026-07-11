<script lang="ts">
	import { ArrowRight, Boxes, Check, Cloud, Code2, Container, Database, GitBranch, GitPullRequestArrow, LockKeyhole, Radio, RotateCcw, Server, ShieldCheck, TerminalSquare } from '@lucide/svelte';
	import MarketingHeader from '$components/MarketingHeader.svelte';
	import ProductPreview from '$components/ProductPreview.svelte';

	const integrations = [
		{ name: 'GitHub', detail: 'OAuth, repositories, push webhooks', icon: GitBranch },
		{ name: 'Cloudflare', detail: 'Tunnel, wildcard domains, edge TLS', icon: Cloud },
		{ name: 'Docker', detail: 'Dockerfile builds and container lifecycle', icon: Container },
		{ name: 'Compose', detail: 'Multi-service applications and networks', icon: Boxes },
		{ name: 'Caddy', detail: 'Dynamic internal reverse proxy routes', icon: GitPullRequestArrow },
		{ name: 'PostgreSQL', detail: 'State, encrypted env vars, shared databases', icon: Database }
	];
	const flowSteps = [
		{ icon: GitBranch, label: 'Push', detail: 'GitHub webhook' },
		{ icon: TerminalSquare, label: 'Build', detail: 'Docker or Compose' },
		{ icon: ShieldCheck, label: 'Route', detail: 'Caddy + Cloudflare' },
		{ icon: Radio, label: 'Observe', detail: 'SSE logs + metrics' }
	];
	const operationalFeatures = [
		{ icon: RotateCcw, label: 'Rollback without rebuilding' },
		{ icon: Radio, label: 'Realtime logs and metrics' },
		{ icon: Database, label: 'Encrypted environment variables' },
		{ icon: ShieldCheck, label: 'GitHub whitelist authentication' }
	];
</script>

<svelte:head><title>MyPaas · Deploy your own cloud</title><meta name="description" content="A self-hosted deployment control plane for Dockerfile, Compose, and static projects." /></svelte:head>

<MarketingHeader active="home" />
<main class="overflow-hidden">
	<section class="hero-shell">
		<div class="mx-auto grid max-w-7xl items-center gap-12 px-4 py-20 sm:px-6 sm:py-24 lg:grid-cols-[0.82fr_1.18fr] lg:px-8 lg:py-28">
			<div class="relative z-10">
				<div class="mb-6 inline-flex items-center gap-2 rounded-full border border-brand-500/25 bg-brand-50 px-3 py-1.5 text-xs font-semibold text-brand-900 dark:bg-brand-500/10 dark:text-brand-100"><Server size={14} aria-hidden="true" /> Your server. Your deployment platform.</div>
				<h1 class="max-w-xl text-balance text-5xl font-bold leading-[1.02] tracking-[-0.035em] text-gray-950 sm:text-6xl dark:text-white">Deploy your own cloud without operating one.</h1>
				<p class="mt-6 max-w-xl text-pretty text-lg leading-8 text-gray-600 dark:text-gray-300">Connect a Git repository, let MyPaas detect Dockerfile or Compose, then build, route, monitor, and recover every project from one quiet control plane.</p>
				<div class="mt-8 flex flex-wrap gap-3">
					<a href="/login" class="inline-flex h-11 items-center gap-2 rounded-md bg-brand-700 px-5 text-sm font-semibold text-white hover:bg-brand-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 dark:bg-brand-500 dark:text-gray-950 dark:hover:bg-brand-100">Open dashboard <ArrowRight size={16} /></a>
					<a href="/docs" class="inline-flex h-11 items-center gap-2 rounded-md border border-gray-300 bg-white px-5 text-sm font-semibold text-gray-800 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100 dark:hover:bg-gray-900"><Code2 size={16} /> Read the docs</a>
				</div>
				<div class="mt-8 flex flex-wrap gap-x-5 gap-y-2 text-xs font-medium text-gray-500 dark:text-gray-400">
					<span class="flex items-center gap-1.5"><Check size={14} class="text-brand-600" /> Dockerfile + Compose</span><span class="flex items-center gap-1.5"><Check size={14} class="text-brand-600" /> Automatic wildcard routing</span><span class="flex items-center gap-1.5"><Check size={14} class="text-brand-600" /> No Kubernetes</span>
				</div>
			</div>
			<div class="relative lg:translate-x-8"><div class="absolute -inset-16 -z-10 bg-[radial-gradient(circle,rgba(20,184,121,0.14),transparent_64%)]"></div><ProductPreview /></div>
		</div>
	</section>

	<section class="border-y border-gray-200 bg-gray-950 text-white dark:border-gray-800" aria-labelledby="flow-heading">
		<div class="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
			<div class="grid gap-10 lg:grid-cols-[0.65fr_1.35fr] lg:items-end"><div><h2 id="flow-heading" class="text-3xl font-bold tracking-[-0.025em]">A deployment path you can explain.</h2><p class="mt-3 max-w-md leading-7 text-gray-400">No hidden buildpack or orchestration layer. Every step maps to infrastructure you already understand.</p></div><div class="grid gap-px overflow-hidden rounded-lg border border-gray-800 bg-gray-800 sm:grid-cols-4">{#each flowSteps as step}<div class="bg-gray-950 p-5"><svelte:component this={step.icon} size={18} class="text-brand-500" /><strong class="mt-4 block text-sm">{step.label}</strong><span class="mt-1 block text-xs text-gray-500">{step.detail}</span></div>{/each}</div></div>
		</div>
	</section>

	<section id="integrations" class="scroll-mt-20 bg-white py-20 dark:bg-gray-950 sm:py-24">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="max-w-2xl"><h2 class="text-3xl font-bold tracking-[-0.025em] text-gray-950 dark:text-white">Integrates with the tools already running your stack.</h2><p class="mt-4 text-lg leading-8 text-gray-600 dark:text-gray-300">MyPaas connects the control plane; it does not replace the proven infrastructure underneath it.</p></div>
			<div class="mt-12 grid border-x border-t border-gray-200 sm:grid-cols-2 lg:grid-cols-3 dark:border-gray-800">{#each integrations as item}<div class="flex gap-4 border-b border-gray-200 p-6 sm:border-r dark:border-gray-800"><div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-brand-50 text-brand-800 dark:bg-brand-500/10 dark:text-brand-300"><svelte:component this={item.icon} size={19} /></div><div><h3 class="font-semibold text-gray-950 dark:text-white">{item.name}</h3><p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-400">{item.detail}</p></div></div>{/each}</div>
		</div>
	</section>

	<section class="border-t border-gray-200 bg-gray-50 py-20 dark:border-gray-800 dark:bg-gray-900/40">
		<div class="mx-auto grid max-w-7xl gap-12 px-4 sm:px-6 lg:grid-cols-2 lg:px-8">
			<div><LockKeyhole size={22} class="text-brand-700 dark:text-brand-400" /><h2 class="mt-5 text-3xl font-bold tracking-[-0.025em] text-gray-950 dark:text-white">Operationally serious, intentionally small.</h2><p class="mt-4 max-w-xl text-lg leading-8 text-gray-600 dark:text-gray-300">Built for one owner and trusted collaborators on a single VM—not as a pretend hyperscaler.</p></div>
			<div class="grid gap-5 sm:grid-cols-2">{#each operationalFeatures as feature}<div class="border-t border-gray-300 pt-4 dark:border-gray-700"><svelte:component this={feature.icon} size={17} class="text-brand-700 dark:text-brand-400" /><h3 class="mt-3 text-sm font-semibold text-gray-950 dark:text-white">{feature.label}</h3></div>{/each}</div>
		</div>
	</section>

	<section class="bg-brand-900 py-16 text-white dark:bg-brand-900"><div class="mx-auto flex max-w-7xl flex-col items-start justify-between gap-6 px-4 sm:px-6 md:flex-row md:items-center lg:px-8"><div><h2 class="text-3xl font-bold tracking-[-0.025em]">One VM is enough to start.</h2><p class="mt-2 text-brand-100">Install MyPaas, connect GitHub, and ship the first project.</p></div><a href="/docs#quick-start" class="inline-flex h-11 items-center gap-2 rounded-md bg-white px-5 text-sm font-semibold text-brand-900 hover:bg-brand-50">View quick start <ArrowRight size={16} /></a></div></section>
</main>

<footer class="border-t border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950"><div class="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-8 text-sm text-gray-500 sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8"><p>MyPaas · Self-hosted deployment control plane.</p><div class="flex gap-5"><a href="/docs" class="hover:text-gray-950 dark:hover:text-white">Documentation</a><a href="https://github.com/nabilrn/mypaas" class="hover:text-gray-950 dark:hover:text-white">GitHub</a></div></div></footer>
