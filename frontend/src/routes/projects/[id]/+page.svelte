<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import StatusBadge from '$components/StatusBadge.svelte';
	import { api } from '$api';
	import type { Deployment, Project } from '$types';

	let project: Project | null = null;
	let deployments: Deployment[] = [];
	let loading = true;

	$: lastDeploy = deployments.find((d) => d.id === project?.activeDeploymentId) ?? deployments[0];
	$: configRows = project
		? [
				['Repository', project.repoUrl],
				['Branch', project.branch],
				['Deploy mode', project.deployMode],
				['App port', String(project.appPort)],
				['Memory', `${project.memoryLimitMb} MB`],
				['CPU', `${project.cpuLimit} cores`]
			]
		: [];

	onMount(() => {
		void load();
		const id = setInterval(load, 3000);
		return () => clearInterval(id);
	});

	async function load() {
		try {
			[project, deployments] = await Promise.all([
				api.projects.get($page.params.id),
				api.deployments.list($page.params.id)
			]);
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{project?.name ?? 'Project'} · MyPaas</title>
</svelte:head>

{#if loading || !project}
	<p class="text-sm text-gray-500 dark:text-gray-400">Loading overview...</p>
{:else}
<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
	<!-- Status card -->
	<div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900">
		<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Status</p>
		<div class="mt-2">
			<StatusBadge status={project.status} pulse />
		</div>
	</div>

	<!-- Uptime -->
	<div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900">
		<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Uptime</p>
		<p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">8h 43m</p>
	</div>

	<!-- Memory -->
	<div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900">
		<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Memory</p>
		<p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">234<span class="text-base font-normal text-gray-500">/{project.memoryLimitMb} MB</span></p>
		<div class="mt-2 h-1.5 rounded-full bg-gray-100 dark:bg-gray-800">
			<div class="h-1.5 rounded-full bg-green-500" style="width: {(234 / project.memoryLimitMb) * 100}%"></div>
		</div>
	</div>

	<!-- CPU -->
	<div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900">
		<p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">CPU</p>
		<p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">12.4<span class="text-base font-normal text-gray-500">%</span></p>
		<div class="mt-2 h-1.5 rounded-full bg-gray-100 dark:bg-gray-800">
			<div class="h-1.5 rounded-full bg-blue-500" style="width: 12.4%"></div>
		</div>
	</div>
</div>
{/if}

{#if project}
<!-- Project details -->
<div class="mt-4 grid gap-4 lg:grid-cols-2">
	<!-- Config -->
	<div class="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
		<h3 class="mb-4 font-semibold text-gray-900 dark:text-white">Configuration</h3>
		<dl class="space-y-3 text-sm">
			{#each configRows as [k, v]}
				<div class="flex justify-between gap-4">
					<dt class="text-gray-500 dark:text-gray-400">{k}</dt>
					<dd class="truncate font-medium text-gray-900 dark:text-white">{v}</dd>
				</div>
			{/each}
		</dl>
	</div>

	<!-- Last deployment -->
	<div class="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
		<h3 class="mb-4 font-semibold text-gray-900 dark:text-white">Last deployment</h3>
		{#if lastDeploy}
			<dl class="space-y-3 text-sm">
				<div class="flex justify-between gap-4">
					<dt class="text-gray-500 dark:text-gray-400">Commit</dt>
					<dd class="font-mono font-medium text-gray-900 dark:text-white">{lastDeploy.commitSha?.slice(0, 8)}</dd>
				</div>
				<div class="flex justify-between gap-4">
					<dt class="text-gray-500 dark:text-gray-400">Message</dt>
					<dd class="truncate font-medium text-gray-900 dark:text-white">{lastDeploy.commitMessage}</dd>
				</div>
				<div class="flex justify-between gap-4">
					<dt class="text-gray-500 dark:text-gray-400">Triggered by</dt>
					<dd class="font-medium text-gray-900 dark:text-white capitalize">{lastDeploy.triggeredBy}</dd>
				</div>
				<div class="flex justify-between gap-4">
					<dt class="text-gray-500 dark:text-gray-400">Status</dt>
					<dd><StatusBadge status={lastDeploy.status} /></dd>
				</div>
			</dl>
		{:else}
			<p class="text-sm text-gray-500 dark:text-gray-400">No deployment yet.</p>
		{/if}
	</div>
</div>
{/if}
