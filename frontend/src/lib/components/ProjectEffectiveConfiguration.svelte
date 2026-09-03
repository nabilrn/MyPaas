<script lang="ts">
	import { Box, ExternalLink, FileText, GitBranch, Globe2 } from '@lucide/svelte';
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
	$: sourceLabel = project.sourceType === 'registry'
		? 'Container registry'
		: project.repoUrl.includes('github.com')
			? 'GitHub'
			: 'Git repository';
	$: sourceSummary = project.sourceType === 'registry'
		? sourceLabel
		: `${sourceLabel}${project.branch ? ` · ${project.branch}` : ''}`;
	$: sourceDetail = project.sourceType === 'registry'
		? project.imageRef || '-'
		: project.repoUrl;
	$: route = publicUrl.replace(/^https?:\/\//, '') || project.subdomain;
</script>

<section class="overflow-hidden rounded-lg border border-gray-200 dark:border-neutral-800">
	<div class="divide-y divide-gray-200 dark:divide-neutral-800">
		<div class="grid gap-3 px-4 py-4 sm:grid-cols-[2rem_14rem_minmax(0,1fr)] sm:items-start">
			<FileText class="mt-0.5 h-4 w-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
			<p class="text-sm text-gray-600 dark:text-gray-400">Project name</p>
			<div class="min-w-0">
				<p class="text-sm font-semibold text-gray-950 dark:text-white">{project.name}</p>
				<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Fixed after creation.</p>
			</div>
		</div>

		<div class="grid gap-3 px-4 py-4 sm:grid-cols-[2rem_14rem_minmax(0,1fr)] sm:items-start">
			<Globe2 class="mt-0.5 h-4 w-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
			<p class="text-sm text-gray-600 dark:text-gray-400">Public URL</p>
			<div class="min-w-0">
				{#if publicUrl}
					<a href={publicUrl} target="_blank" rel="noopener" class="app-focus inline-flex max-w-full items-center gap-1.5 font-mono text-sm font-medium text-gray-950 hover:underline dark:text-white">
						<span class="truncate">{route}</span>
						<ExternalLink class="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					</a>
				{:else}
					<p class="truncate font-mono text-sm font-medium text-gray-950 dark:text-white">{route}</p>
				{/if}
				<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Open in browser to visit your project.</p>
			</div>
		</div>

		<div class="grid gap-3 px-4 py-4 sm:grid-cols-[2rem_14rem_minmax(0,1fr)] sm:items-start">
			<Box class="mt-0.5 h-4 w-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
			<p class="text-sm text-gray-600 dark:text-gray-400">Deployment type</p>
			<div class="min-w-0">
				<p class="text-sm font-semibold text-gray-950 dark:text-white">{deploymentType}</p>
				<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{deploymentDescription}</p>
			</div>
		</div>

		<div class="grid gap-3 px-4 py-4 sm:grid-cols-[2rem_14rem_minmax(0,1fr)] sm:items-start">
			<GitBranch class="mt-0.5 h-4 w-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
			<p class="text-sm text-gray-600 dark:text-gray-400">Source</p>
			<div class="min-w-0">
				<p class="text-sm font-semibold text-gray-950 dark:text-white">{sourceSummary}</p>
				<p class="mt-1 truncate font-mono text-sm text-gray-500 dark:text-gray-400" title={sourceDetail}>{sourceDetail}</p>
			</div>
		</div>
	</div>
</section>
