<script lang="ts">
	import { onMount } from 'svelte';
	import { afterNavigate, goto } from '$app/navigation';
	import { page } from '$app/stores';
	import Breadcrumbs from '$components/Breadcrumbs.svelte';
	import DeployControlPanel from '$components/DeployControlPanel.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { Project, ProjectStatus } from '$types';
	import { projectHost, projectURL } from '$lib/utils/urls';

	const terminalProjectStatuses = new Set<ProjectStatus>(['running', 'stopped', 'crashed', 'pending']);

	let project: Project | null = null;
	let loading = true;
	let error = '';
	let pendingAction: 'stop' | 'restart' | 'deploy' | null = null;
	let optimisticHref: string | null = null;
	let projectRefreshInFlight = false;
	let stream: EventSource | null = null;
	let lastStreamStatus: ProjectStatus | null = null;

	const tabs = [
		{ label: 'Overview', href: '' },
		{ label: 'Deployments', href: '/deployments' },
		{ label: 'Logs', href: '/logs' },
		{ label: 'Metrics', href: '/metrics' },
		{ label: 'Environment', href: '/env' },
		{ label: 'Database', href: '/database' },
		{ label: 'Settings', href: '/settings' }
	];

	$: base = `/projects/${$page.params.id}`;
	$: pathname = $page.url.pathname;
	$: currentPath = normalizePath(pathname);
	$: routeActiveHref =
		tabs
			.slice()
			.reverse()
			.find((t) => isTabActive(t.href, currentPath, base))?.href ?? '';
	$: activeHref = optimisticHref ?? routeActiveHref;
	$: activeTab = tabs.find((tab) => tab.href === activeHref) ?? tabs[0];
	$: breadcrumbs = project
		? [{ label: 'Projects', href: '/projects' }, { label: project.name, href: activeHref ? base : undefined }, ...(activeHref ? [{ label: activeTab.label }] : [])]
		: [{ label: 'Projects', href: '/projects' }, { label: 'Project' }];
	$: publicProjectHost = project ? projectHost(project.subdomain, $page.url.hostname) : '';
	$: publicProjectURL = project ? projectURL(project.subdomain, $page.url.protocol, $page.url.hostname) : '';
	$: logsHref = `${base}/logs`;

	afterNavigate(() => {
		optimisticHref = null;
	});

	function normalizePath(value: string) {
		return value.length > 1 && value.endsWith('/') ? value.slice(0, -1) : value;
	}

	function tabPath(href: string, currentBase: string) {
		return normalizePath(`${currentBase}${href}`);
	}

	function isTabActive(href: string, currentPathname: string, currentBase: string) {
		const targetPath = tabPath(href, currentBase);
		return href === '' ? currentPathname === targetPath : currentPathname === targetPath || currentPathname.startsWith(`${targetPath}/`);
	}

	function tabClass(active: boolean) {
		return `whitespace-nowrap rounded-md px-3 py-1.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950 ${
			active
				? 'bg-brand-700 text-white shadow-sm dark:bg-brand-500 dark:text-gray-950'
				: 'text-gray-500 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white'
		}`;
	}

	function handleTabClick(href: string) {
		if (href === activeHref) return;
		optimisticHref = href;
	}

	onMount(() => {
		void loadProject();
		connectProjectStream();

		return () => {
			stream?.close();
			stream = null;
		};
	});

	function connectProjectStream() {
		stream?.close();
		stream = new EventSource(`/api/projects/${$page.params.id}/stream`, { withCredentials: true });
		stream.addEventListener('status', handleStatusEvent);
	}

	function handleStatusEvent(event: MessageEvent) {
		try {
			const parsed = JSON.parse(event.data) as { status?: string };
			if (parsed.status === 'deleted') {
				stream?.close();
				project = null;
				error = 'Project not found';
				return;
			}
			if (!isProjectStatus(parsed.status)) return;

			const previousStatus = project?.status ?? lastStreamStatus;
			lastStreamStatus = parsed.status;
			if (project) {
				project = { ...project, status: parsed.status };
			}
			if (previousStatus !== parsed.status && terminalProjectStatuses.has(parsed.status)) {
				void loadProject(true);
			}
		} catch {
			// Ignore malformed stream events; EventSource will keep the connection alive.
		}
	}

	function isProjectStatus(status: string | undefined): status is ProjectStatus {
		return status === 'pending' || status === 'running' || status === 'stopped' || status === 'crashed' || status === 'building';
	}

	async function loadProject(background = false) {
		if (projectRefreshInFlight) return;
		projectRefreshInFlight = true;
		if (!background || !project) {
			loading = true;
		}
		if (!background || !project) {
			error = '';
		}
		try {
			project = await api.projects.get($page.params.id ?? '');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load project';
			if (!background || !project) {
				error = message;
			}
		} finally {
			if (!background || !project) {
				loading = false;
			}
			projectRefreshInFlight = false;
		}
	}

	async function handleStop() {
		if (!project || pendingAction) return;
		pendingAction = 'stop';
		try {
			await api.projects.stop(project.id);
			toast.success(`Stopped ${project.name}`);
			await loadProject(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to stop project');
		} finally {
			pendingAction = null;
		}
	}

	async function handleRestart() {
		if (!project || pendingAction) return;
		pendingAction = 'restart';
		try {
			await api.projects.restart(project.id);
			toast.success(`Restarted ${project.name}`);
			await loadProject(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to restart project');
		} finally {
			pendingAction = null;
		}
	}

	async function handleDeploy() {
		if (!project || pendingAction) return;
		pendingAction = 'deploy';
		try {
			const deployment = await api.projects.deploy(project.id);
			toast.success(`Deployment queued for ${project.name}`);
			await goto(`/projects/${project.id}/deployments?focus=${encodeURIComponent(deployment.id)}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to deploy project');
		} finally {
			pendingAction = null;
		}
	}
</script>

<div class="page-shell py-5">
	<Breadcrumbs items={breadcrumbs} />

	{#if loading}
		<div class="surface p-5">
			<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div class="min-w-0 flex-1">
					<div class="h-5 w-48 animate-pulse rounded bg-gray-200 dark:bg-gray-800"></div>
					<div class="mt-3 h-3 w-full max-w-xl animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
				</div>
				<div class="h-9 w-40 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>
			</div>
			<div class="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
				{#each [1, 2, 3, 4] as _}
					<div class="h-12 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>
				{/each}
			</div>
		</div>
	{:else if error || !project}
		<div class="surface overflow-hidden">
			<ErrorState title="Could not load project" message={error || 'Project not found'} on:retry={() => void loadProject()} />
		</div>
	{:else}
		<DeployControlPanel {project} {publicProjectHost} {publicProjectURL} {logsHref} {pendingAction} on:stop={handleStop} on:restart={handleRestart} on:deploy={handleDeploy}>
			<nav class="scrollbar-thin flex gap-1 overflow-x-auto border-t border-gray-100 px-3 py-2 dark:border-gray-800" data-sveltekit-preload-data="tap" aria-label={`${project.name} sections`}>
				{#each tabs as tab}
					<a href={base + tab.href} on:click={() => handleTabClick(tab.href)} aria-current={activeHref === tab.href ? 'page' : undefined} class={tabClass(activeHref === tab.href)}>
						{tab.label}
					</a>
				{/each}
			</nav>
		</DeployControlPanel>

		<div class="py-5">
			<slot />
		</div>
	{/if}
</div>
