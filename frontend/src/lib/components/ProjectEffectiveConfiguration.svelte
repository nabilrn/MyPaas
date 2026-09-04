<script lang="ts">
	import { ExternalLink } from '@lucide/svelte';
	import type { Project } from '$types';

	export let project: Project;
	export let publicUrl = '';

	$: deploymentType = project.deployMode === 'compose'
		? 'Docker Compose'
		: project.deployMode === 'dockerfile'
			? 'Dockerfile'
			: project.deployMode === 'static'
				? 'Static site'
				: 'Container image';
	$: deploymentDescription = project.deployMode === 'compose'
		? 'Deployed using Docker Compose.'
		: project.deployMode === 'dockerfile'
			? 'Built and deployed from a Dockerfile.'
			: project.deployMode === 'static'
				? 'Published as a static site.'
				: 'Deployed from a container image.';
	$: route = publicUrl.replace(/^https?:\/\//, '') || project.subdomain;
</script>

<section class="-mx-4 border-y border-[color:var(--workspace-divider)]">
	<div class="divide-y divide-[color:var(--workspace-divider)]">
		<div class="grid gap-2 px-4 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-start">
			<p class="text-sm text-gray-500 dark:text-gray-400">Project name</p>
			<div class="min-w-0">
				<p class="text-sm font-semibold text-gray-950 dark:text-white">{project.name}</p>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Fixed after creation.</p>
			</div>
		</div>

		<div class="grid gap-2 px-4 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-start">
			<p class="text-sm text-gray-500 dark:text-gray-400">Public URL</p>
			<div class="min-w-0">
				{#if publicUrl}
					<a href={publicUrl} target="_blank" rel="noopener" class="app-focus inline-flex max-w-full items-center gap-1.5 font-mono text-sm font-medium text-gray-950 hover:underline dark:text-white">
						<span class="truncate">{route}</span>
						<ExternalLink class="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					</a>
				{:else}
					<p class="truncate font-mono text-sm font-medium text-gray-950 dark:text-white">{route}</p>
				{/if}
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Open in browser to visit your project.</p>
			</div>
		</div>

		<div class="grid gap-2 px-4 py-3 sm:grid-cols-[11rem_minmax(0,1fr)] sm:items-start">
			<p class="text-sm text-gray-500 dark:text-gray-400">Deployment type</p>
			<div class="min-w-0">
				<p class="text-sm font-semibold text-gray-950 dark:text-white">{deploymentType}</p>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{deploymentDescription}</p>
			</div>
		</div>
	</div>
</section>
