<script lang="ts">
	import ProjectStatus from './ProjectStatus.svelte';
	import { deriveProjectOperationalState } from '$lib/utils/project-operational-state';
	import type { Project } from '$types';

	export let project: Project;
	export let publicUrl = '';

	$: operationalState = deriveProjectOperationalState({ project, latestDeployment: undefined });
	$: desiredLabel = operationalState.desired === 'stopped' ? 'Stopped' : 'Running';
	$: runtime = project.deployMode === 'compose'
		? `Docker Compose${project.mainService ? ` · ${project.mainService}` : ''}`
		: project.deployMode === 'dockerfile'
			? 'Dockerfile'
			: project.deployMode === 'static'
				? 'Static site'
				: 'Container image';
	$: source = project.sourceType === 'registry'
		? project.imageRef || '-'
		: `${project.repoUrl}${project.branch ? ` · ${project.branch}` : ''}`;
	$: route = publicUrl.replace(/^https?:\/\//, '') || project.subdomain;
</script>

<section class="surface overflow-hidden">
	<div class="panel-header">
		<h2 class="panel-title">Effective configuration</h2>
		<p class="panel-description">Saved control-plane configuration. Editing below does not recreate the currently running runtime.</p>
	</div>
	<div class="grid gap-px bg-gray-100 dark:bg-neutral-800 sm:grid-cols-2 xl:grid-cols-3">
		<div class="min-w-0 bg-white p-4 dark:bg-neutral-950">
			<p class="metric-label">Source</p>
			<p class="mt-1 truncate font-mono text-sm text-gray-950 dark:text-white" title={source}>{source}</p>
		</div>
		<div class="min-w-0 bg-white p-4 dark:bg-neutral-950">
			<p class="metric-label">Runtime</p>
			<p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">{runtime}</p>
			<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{project.deployMode === 'static' ? 'Caddy-managed' : `Container port ${project.appPort}`}</p>
		</div>
		<div class="min-w-0 bg-white p-4 dark:bg-neutral-950">
			<p class="metric-label">Resources</p>
			<p class="metric-value mt-1 text-sm font-medium text-gray-950 dark:text-white">{project.memoryLimitMb} MB · {project.cpuLimit} CPU</p>
			<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{project.resourceProfile} profile</p>
		</div>
		<div class="min-w-0 bg-white p-4 dark:bg-neutral-950">
			<p class="metric-label">Public route</p>
			<p class="mt-1 truncate font-mono text-sm text-gray-950 dark:text-white" title={route}>{route}</p>
		</div>
		<div class="min-w-0 bg-white p-4 dark:bg-neutral-950">
			<p class="metric-label">Base directory</p>
			<p class="mt-1 truncate font-mono text-sm text-gray-950 dark:text-white">{project.baseDirectory || 'Repository root'}</p>
		</div>
		<div class="min-w-0 bg-white p-4 dark:bg-neutral-950">
			<p class="metric-label">Desired state</p>
			<div class="mt-1"><ProjectStatus status={operationalState.desired} label={desiredLabel} tone="neutral" /></div>
		</div>
	</div>
</section>
