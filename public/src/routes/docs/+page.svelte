<script lang="ts">
	import { ArrowRight, BookOpen, Box, Cloud, Container, Database, GitBranch, KeyRound, Network, Radio, Server, Terminal } from '@lucide/svelte';
	import MarketingHeader from '$components/MarketingHeader.svelte';
	const sections = [['quick-start','Quick start'],['deploy-modes','Deploy modes'],['integrations','Integrations'],['operations','Operations'],['security','Security']];
	const deployModes = [
		{ icon: Container, name: 'Docker Compose', config: 'docker-compose.yml or compose.yml', detail: 'Multi-service apps, private networks, databases, and queues.' },
		{ icon: Box, name: 'Dockerfile', config: 'Dockerfile', detail: 'A single application container with controlled port and resource limits.' },
		{ icon: Server, name: 'Static', config: 'dist, build, public, or index.html', detail: 'Files served directly through Caddy without an application container.' }
	];
	const integrationItems = [
		{ icon: GitBranch, name: 'GitHub', detail: 'OAuth login, repository inspection, commit metadata, and signed push webhooks.' },
		{ icon: Cloud, name: 'Cloudflare Tunnel', detail: 'Wildcard public hostname and edge TLS without exposing the VM address.' },
		{ icon: Container, name: 'Docker + Compose', detail: 'Image builds, container lifecycle, service networks, resource limits, and log collection.' },
		{ icon: Network, name: 'Caddy', detail: 'Dynamic reverse-proxy and static-file routes managed through the Admin API.' },
		{ icon: Database, name: 'PostgreSQL', detail: 'MyPaas state, audit history, port allocation, encrypted env metadata, and optional shared project databases.' },
		{ icon: Radio, name: 'Server-Sent Events', detail: 'One authenticated stream per project for status, deployments, logs, and metrics.' }
	];
	const securityItems = [
		{ icon: KeyRound, name: 'Credentials', detail: 'JWT, OAuth tokens, webhook secrets, and project env values stay out of logs.' },
		{ icon: Database, name: 'Environment values', detail: 'User values are encrypted using AES-256-GCM before persistence.' },
		{ icon: Server, name: 'Runtime exposure', detail: 'Project ports bind to the configured private host and route through Caddy.' }
	];
</script>

<svelte:head><title>Documentation · MyPaas</title><meta name="description" content="Install, configure, and operate MyPaas on your own server." /></svelte:head>
<MarketingHeader active="docs" />

<div class="mx-auto grid max-w-7xl lg:grid-cols-[15rem_minmax(0,1fr)]">
	<aside class="hidden border-r border-gray-200 px-6 py-10 dark:border-gray-800 lg:block"><nav class="sticky top-24" aria-label="Documentation"><p class="mb-3 text-xs font-semibold text-gray-950 dark:text-white">Documentation</p>{#each sections as section}<a href="#{section[0]}" class="block rounded-md px-3 py-2 text-sm text-gray-600 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-gray-900 dark:hover:text-white">{section[1]}</a>{/each}</nav></aside>
	<main class="min-w-0 px-4 py-12 sm:px-8 lg:px-12 lg:py-16">
		<div class="max-w-3xl">
			<div class="flex items-center gap-2 text-sm font-semibold text-brand-700 dark:text-brand-500"><BookOpen size={17} /> MyPaas documentation</div>
			<h1 class="mt-4 text-balance text-4xl font-bold tracking-[-0.03em] text-gray-950 sm:text-5xl dark:text-white">Build, deploy, and operate projects on your own VM.</h1>
			<p class="mt-5 text-lg leading-8 text-gray-600 dark:text-gray-300">This guide covers the supported deployment path from repository connection through routing, monitoring, rollback, and backup.</p>
		</div>

		<section id="quick-start" class="doc-section"><h2>Quick start</h2><p>Run the installer on a clean Linux VM with Docker and Docker Compose available. The setup wizard creates secrets and walks through GitHub OAuth and Cloudflare Tunnel configuration.</p><div class="code-block"><div class="code-title"><Terminal size={14} /> VM shell</div><code>curl -fsSL https://raw.githubusercontent.com/nabilrn/mypaas/main/scripts/bootstrap.sh | bash</code></div><p>After setup, open the dashboard hostname configured for the Cloudflare Tunnel and sign in using a whitelisted GitHub account.</p></section>

		<section id="deploy-modes" class="doc-section"><h2>Deploy modes</h2><p>Detection follows a predictable priority. Compose wins when both configuration types exist.</p><div class="doc-grid">{#each deployModes as mode}<article><svelte:component this={mode.icon} size={19} /><h3>{mode.name}</h3><code>{mode.config}</code><p>{mode.detail}</p></article>{/each}</div></section>

		<section id="integrations" class="doc-section"><h2>Integrations</h2><p>MyPaas is deliberately built around infrastructure with clear operational boundaries.</p><div class="divide-y divide-gray-200 border-y border-gray-200 dark:divide-gray-800 dark:border-gray-800">{#each integrationItems as item}<div class="flex gap-4 py-5"><div class="mt-0.5 text-brand-700 dark:text-brand-500"><svelte:component this={item.icon} size={19} /></div><div><h3 class="font-semibold text-gray-950 dark:text-white">{item.name}</h3><p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-400">{item.detail}</p></div></div>{/each}</div></section>

		<section id="operations" class="doc-section"><h2>Operations</h2><p>Every deploy is asynchronous and serialized per project. MyPaas allows two global deployments by default, records build output, then updates routing only after the replacement runtime is ready.</p><ol class="steps"><li><span>1</span><div><strong>Connect repository</strong><p>Select a branch and inspect the detected deployment plan.</p></div></li><li><span>2</span><div><strong>Configure runtime</strong><p>Review app port, service, resource profile, and discovered environment variables.</p></div></li><li><span>3</span><div><strong>Deploy and observe</strong><p>Follow build logs, runtime logs, metrics, and status through SSE.</p></div></li><li><span>4</span><div><strong>Recover safely</strong><p>Restart, stop, redeploy, or rollback to a previous successful image.</p></div></li></ol></section>

		<section id="security" class="doc-section"><h2>Security model</h2><div class="doc-grid">{#each securityItems as item}<article><svelte:component this={item.icon} size={19} /><h3>{item.name}</h3><p>{item.detail}</p></article>{/each}</div></section>

		<div class="mt-16 flex flex-col justify-between gap-4 border-t border-gray-200 pt-8 sm:flex-row sm:items-center dark:border-gray-800"><p class="text-sm text-gray-500">Need the complete product requirements and architecture?</p><a href="https://github.com/nabilrn/mypaas/tree/main/docs" class="inline-flex items-center gap-2 text-sm font-semibold text-brand-700 hover:text-brand-900 dark:text-brand-500">Browse repository docs <ArrowRight size={15} /></a></div>
	</main>
</div>
