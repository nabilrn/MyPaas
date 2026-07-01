<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { Project } from '$types';
	import { projectHost, webhookURL } from '$lib/utils/urls';

	let project: Project | null = null;

	let name        = '';
	let branch      = '';
	let appPort     = 3000;
	let memoryMb    = 512;
	let cpuLimit    = 0.5;
	let deleteInput = '';
	let regeneratedSecret = '';
	let regeneratingSecret = false;

	$: nameChanged = project && (name !== project.name || branch !== project.branch ||
	                 appPort !== project.appPort ||
	                 memoryMb !== project.memoryLimitMb || cpuLimit !== project.cpuLimit);
	$: changedProjectHost = projectHost(name || 'your-app', $page.url.hostname);
	$: publicWebhookURL = project ? webhookURL(project.id, $page.url.origin) : '';

	onMount(load);

	async function load() {
		project = await api.projects.get($page.params.id);
		name = project.name;
		branch = project.branch;
		appPort = project.appPort;
		memoryMb = project.memoryLimitMb;
		cpuLimit = project.cpuLimit;
	}

	async function handleSave() {
		if (!project) return;
		try {
			project = await api.projects.update(project.id, {
				name,
				branch,
				appPort: Number(appPort),
				memoryLimitMb: Number(memoryMb),
				cpuLimit: Number(cpuLimit)
			});
			toast.success('Settings saved');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save settings');
		}
	}

	async function handleRegenerateSecret() {
		if (!project || !window.confirm('Regenerate webhook secret? Existing GitHub webhook signatures will stop working until you update the secret.')) return;
		regeneratingSecret = true;
		try {
			const result = await api.projects.regenerateWebhookSecret(project.id);
			regeneratedSecret = result.webhookSecret;
			toast.success('Webhook secret regenerated');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to regenerate webhook secret');
		} finally {
			regeneratingSecret = false;
		}
	}

	function copyWebhookURL(projectId: string) {
		navigator.clipboard?.writeText(webhookURL(projectId, $page.url.origin));
		toast.success('Copied!');
	}

	async function handleDelete() {
		if (!project || deleteInput !== project.name) return;
		try {
			await api.projects.delete(project.id);
			toast.success('Project deleted');
			await goto('/projects');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete project');
		}
	}
</script>

<svelte:head>
	<title>Settings · MyPaas</title>
</svelte:head>

{#if !project}
	<p class="text-sm text-gray-500 dark:text-gray-400">Loading settings...</p>
{:else}
<div class="max-w-2xl space-y-6">
	<!-- General settings -->
	<div class="rounded-xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
		<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
			<h2 class="font-semibold text-gray-900 dark:text-white">General</h2>
		</div>
		<div class="space-y-4 p-5">
			<div>
				<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="pname">
					Project name
				</label>
				<input
					id="pname"
					type="text"
					bind:value={name}
					class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
						   dark:border-gray-700 dark:bg-gray-800 dark:text-white"
				/>
				{#if name !== project.name}
					<p class="mt-1 text-xs text-amber-600 dark:text-amber-400">
						Subdomain will change to <span class="font-mono">{changedProjectHost}</span>
					</p>
				{/if}
			</div>
			<div>
				<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="pbranch">
					Deploy branch
				</label>
				<input
					id="pbranch"
					type="text"
					bind:value={branch}
					class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
						   dark:border-gray-700 dark:bg-gray-800 dark:text-white"
				/>
			</div>
			<div>
				<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="appPort">
					App port
				</label>
				<input
					id="appPort"
					type="number"
					min="1"
					max="65535"
					bind:value={appPort}
					class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
						   dark:border-gray-700 dark:bg-gray-800 dark:text-white"
				/>
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					Use 80 for nginx/static sites.
				</p>
			</div>
		</div>
	</div>

	<!-- Resource limits -->
	<div class="rounded-xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
		<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
			<h2 class="font-semibold text-gray-900 dark:text-white">Resource limits</h2>
		</div>
		<div class="grid gap-4 p-5 sm:grid-cols-2">
			<div>
				<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="mem">
					Memory (MB)
				</label>
				<select
					id="mem"
					bind:value={memoryMb}
					class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"
				>
					{#each [256, 512, 1024, 2048] as m}
						<option value={m}>{m} MB</option>
					{/each}
				</select>
			</div>
			<div>
				<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="cpu">
					CPU (cores)
				</label>
				<select
					id="cpu"
					bind:value={cpuLimit}
					class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"
				>
					{#each [0.25, 0.5, 1, 2] as c}
						<option value={c}>{c} core{c !== 1 ? 's' : ''}</option>
					{/each}
				</select>
			</div>
		</div>
		{#if nameChanged}
			<div class="border-t border-gray-100 px-5 py-3 dark:border-gray-800">
				<button
					on:click={handleSave}
					class="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
				>
					Save changes
				</button>
			</div>
		{/if}
	</div>

	<!-- Webhook -->
	<div class="rounded-xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
		<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
			<h2 class="font-semibold text-gray-900 dark:text-white">Webhook</h2>
		</div>
		<div class="space-y-3 p-5 text-sm">
			<div>
				<p class="mb-1 font-medium text-gray-700 dark:text-gray-300">Webhook URL</p>
				<div class="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-800">
					<code class="flex-1 text-xs text-gray-600 dark:text-gray-400">
						{publicWebhookURL}
					</code>
					<button
						on:click={() => copyWebhookURL(project?.id ?? '')}
						class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
						aria-label="Copy webhook URL"
					>
						<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
						</svg>
					</button>
				</div>
			</div>
			<div>
				<p class="mb-1 font-medium text-gray-700 dark:text-gray-300">Webhook secret</p>
				<p class="mb-2 text-xs text-gray-500 dark:text-gray-400">Add this as <code class="font-mono">MYPAAS_WEBHOOK_SECRET</code> in your GitHub repo settings.</p>
				<button
					on:click={handleRegenerateSecret}
					disabled={regeneratingSecret}
					class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700
						   hover:bg-gray-50 disabled:opacity-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
				>
					{regeneratingSecret ? 'Regenerating...' : 'Regenerate secret'}
				</button>
				{#if regeneratedSecret}
					<div class="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 dark:border-amber-900/60 dark:bg-amber-950/20">
						<p class="mb-1 text-xs font-medium text-amber-800 dark:text-amber-300">New secret</p>
						<code class="break-all text-xs text-amber-900 dark:text-amber-200">{regeneratedSecret}</code>
					</div>
				{/if}
			</div>
		</div>
	</div>

	<!-- Danger zone -->
	<div class="rounded-xl border border-red-200 bg-white dark:border-red-900/50 dark:bg-gray-900">
		<div class="border-b border-red-100 px-5 py-4 dark:border-red-900/50">
			<h2 class="font-semibold text-red-700 dark:text-red-400">Danger zone</h2>
		</div>
		<div class="p-5">
			<p class="mb-3 text-sm text-gray-600 dark:text-gray-400">
				Deleting <strong class="text-gray-900 dark:text-white">{project.name}</strong> will stop all containers,
				remove routing, and release ports. This action <strong>cannot be undone</strong>.
			</p>
			<p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
				Type <code class="font-mono text-red-600 dark:text-red-400">{project.name}</code> to confirm:
			</p>
			<div class="flex gap-2">
				<input
					type="text"
					bind:value={deleteInput}
					placeholder={project.name}
					class="flex-1 rounded-lg border border-red-300 px-3 py-2 text-sm focus:border-red-500 focus:outline-none focus:ring-1 focus:ring-red-500
						   dark:border-red-900 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
				/>
				<button
					on:click={handleDelete}
					disabled={deleteInput !== project.name}
					class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white
						   hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-40"
				>
					Delete project
				</button>
			</div>
		</div>
	</div>
</div>
{/if}
