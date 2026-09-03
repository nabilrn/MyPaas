<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import DeployControlPanel from '$components/DeployControlPanel.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import ProjectDetailSidebar from '$components/ProjectDetailSidebar.svelte';
	import { api } from '$api';
	import { beginMainContentLoading } from '$stores/main-loading';
	import { clearShellContext, setShellContext } from '$stores/shell-context';
	import { toast } from '$stores/toast';
	import {
		appendProjectStreamLog,
		projectStreamConnection,
		projectStreamLogs,
		projectStreamMetrics,
		resetProjectStreamState,
		setProjectStreamReconnect
	} from '$stores/project-stream';
	import { projectStreamTopics } from '$lib/utils/project-stream-topics';
	import type { Deployment, MetricsSnapshot, Project, ProjectStatus } from '$types';
	import { projectHost, projectURL } from '$lib/utils/urls';

	const terminalProjectStatuses = new Set<ProjectStatus>(['running', 'stopped', 'crashed', 'pending']);

	let project: Project | null = null;
	let latestDeployment: Deployment | null | undefined = undefined;
	let loading = true;
	let error = '';
	let pendingAction: 'start' | 'stop' | 'restart' | 'deploy' | null = null;
	let projectRefreshInFlight = false;
	let stream: EventSource | null = null;
	let lastStreamStatus: ProjectStatus | null = null;
	let mounted = false;
	let activeStreamKey = '';

	$: publicProjectHost = project ? projectHost(project.subdomain, $page.url.hostname) : '';
	$: publicProjectURL = project ? projectURL(project.subdomain, $page.url.protocol, $page.url.hostname) : '';
	$: setShellContext(project ? { projectId: project.id, projectName: project.name } : {});
	$: desiredTopics = project ? projectStreamTopics($page.url.pathname, project.id, project.deployMode) : 'status';
	$: desiredStreamKey = `${$page.params.id}:${desiredTopics}`;
	$: databaseWorkspace = $page.url.pathname.startsWith(`/projects/${$page.params.id}/database`);
	$: if (mounted && project && desiredStreamKey !== activeStreamKey) connectProjectStream();

	onMount(() => {
		mounted = true;
		resetProjectStreamState();
		setProjectStreamReconnect(() => connectProjectStream(true));
		void loadProject();

		return () => {
			mounted = false;
			stream?.close();
			stream = null;
			activeStreamKey = '';
			setProjectStreamReconnect(null);
			resetProjectStreamState();
			clearShellContext();
		};
	});

	function connectProjectStream(force = false) {
		if (!mounted || !project) return;
		const topics = projectStreamTopics($page.url.pathname, project.id, project.deployMode);
		const key = `${project.id}:${topics}`;
		if (!force && stream && key === activeStreamKey) return;

		stream?.close();
		stream = null;
		activeStreamKey = key;
		projectStreamConnection.set('connecting');
		if (!topics.split(',').includes('metrics')) projectStreamMetrics.set(null);
		if (!topics.split(',').includes('logs')) projectStreamLogs.set([]);

		stream = new EventSource(`/api/projects/${project.id}/stream?topics=${encodeURIComponent(topics)}`, { withCredentials: true });
		stream.addEventListener('open', () => projectStreamConnection.set('open'));
		stream.addEventListener('error', () => projectStreamConnection.set('reconnecting'));
		stream.addEventListener('status', handleStatusEvent);
		stream.addEventListener('metrics', handleMetricsEvent);
		stream.addEventListener('log', handleLogEvent);
		stream.addEventListener('deployment-log', handleLogEvent);
	}

	function handleMetricsEvent(event: MessageEvent) {
		try {
			const parsed = JSON.parse(event.data) as MetricsSnapshot;
			if (!Array.isArray(parsed.items)) return;
			projectStreamMetrics.set(parsed);
		} catch {
			// EventSource reconnects independently; malformed samples are ignored.
		}
	}

	function handleLogEvent(event: MessageEvent) {
		try {
			const parsed = JSON.parse(event.data) as { service?: string; line?: string; timestamp?: string };
			if (!parsed.line) return;
			appendProjectStreamLog({
				service: parsed.service || 'app',
				line: parsed.line,
				timestamp: parsed.timestamp || new Date().toISOString()
			});
		} catch {
			// Ignore malformed stream events.
		}
	}

	function handleStatusEvent(event: MessageEvent) {
		try {
			const parsed = JSON.parse(event.data) as { status?: string };
			if (parsed.status === 'deleted') {
				stream?.close();
				project = null;
				latestDeployment = undefined;
				error = 'Project not found';
				return;
			}
			if (!isProjectStatus(parsed.status)) return;

			const previousStatus = project?.status ?? lastStreamStatus;
			lastStreamStatus = parsed.status;
			if (project) project = { ...project, status: parsed.status };
			if (previousStatus !== parsed.status && terminalProjectStatuses.has(parsed.status)) void loadProject(true);
		} catch {
			// Ignore malformed stream events; EventSource keeps the connection alive.
		}
	}

	function isProjectStatus(status: string | undefined): status is ProjectStatus {
		return status === 'pending' || status === 'running' || status === 'stopped' || status === 'crashed' || status === 'building';
	}

	async function loadProject(background = false) {
		if (projectRefreshInFlight) return;
		projectRefreshInFlight = true;
		const foreground = !background && !project;
		const finishMainLoading = foreground ? beginMainContentLoading() : null;
		if (!background || !project) {
			loading = true;
			error = '';
		}
		try {
			const projectId = $page.params.id ?? '';
			const [projectResult, deploymentResult] = await Promise.allSettled([
				api.projects.get(projectId),
				api.deployments.list(projectId, 0, 1)
			]);
			if (projectResult.status === 'rejected') throw projectResult.reason;
			project = projectResult.value;
			latestDeployment = deploymentResult.status === 'fulfilled' ? (deploymentResult.value[0] ?? null) : undefined;
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load project';
			if (!background || !project) error = message;
		} finally {
			if (!background || !project) loading = false;
			finishMainLoading?.();
			projectRefreshInFlight = false;
		}
	}

	async function handleStart() {
		if (!project || pendingAction) return;
		pendingAction = 'start';
		try {
			await api.projects.start(project.id);
			toast.success(`Started ${project.name}`);
			await loadProject(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to start project');
		} finally {
			pendingAction = null;
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
			latestDeployment = deployment;
			await goto(`/projects/${project.id}/deployments?focus=${encodeURIComponent(deployment.id)}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to deploy project');
		} finally {
			pendingAction = null;
		}
	}
</script>

<div class="page-shell">
	{#if !loading && (error || !project)}
		<div class="workspace-section">
			<ErrorState title="Could not load project" message={error || 'Project not found'} on:retry={() => void loadProject()} />
		</div>
	{:else if project}
		<div class="grid min-h-[calc(100vh-3.5rem)] lg:grid-cols-[12rem_minmax(0,1fr)]">
			<aside class="border-b border-[color:var(--workspace-divider)] px-3 py-4 lg:border-b-0 lg:border-r">
				<ProjectDetailSidebar projectId={project.id} />
			</aside>

			<main class="min-w-0 px-3.5 py-3">
				<div class="space-y-3">
					{#if !databaseWorkspace}
						<DeployControlPanel
							{project}
							{latestDeployment}
							{publicProjectHost}
							{publicProjectURL}
							{pendingAction}
							on:start={handleStart}
							on:stop={handleStop}
							on:restart={handleRestart}
							on:deploy={handleDeploy}
						/>
					{/if}

					<div class="project-detail-content min-w-0">
						<slot />
					</div>
				</div>
			</main>
		</div>
	{/if}
</div>

<style>
	:global(.project-detail-content > .page-shell) {
		width: 100%;
		max-width: none;
		margin-inline: 0;
		padding: 0;
	}
</style>
