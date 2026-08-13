<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import DeployControlPanel from '$components/DeployControlPanel.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import { api } from '$api';
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
	import type { MetricsSnapshot, Project, ProjectStatus } from '$types';
	import { projectHost, projectURL } from '$lib/utils/urls';

	const terminalProjectStatuses = new Set<ProjectStatus>(['running', 'stopped', 'crashed', 'pending']);

	let project: Project | null = null;
	let loading = true;
	let error = '';
	let pendingAction: 'stop' | 'restart' | 'deploy' | null = null;
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
				error = 'Project not found';
				return;
			}
			if (!isProjectStatus(parsed.status)) return;

			const previousStatus = project?.status ?? lastStreamStatus;
			lastStreamStatus = parsed.status;
			if (project) project = { ...project, status: parsed.status };
			if (previousStatus !== parsed.status && terminalProjectStatuses.has(parsed.status)) {
				void loadProject(true);
			}
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
		if (!background || !project) {
			loading = true;
			error = '';
		}
		try {
			project = await api.projects.get($page.params.id ?? '');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load project';
			if (!background || !project) error = message;
		} finally {
			if (!background || !project) loading = false;
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
	{#if loading}
		<div class="control-panel p-5">
			<div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
				<div class="min-w-0 flex-1">
					<div class="h-7 w-48 animate-pulse rounded bg-gray-200 dark:bg-gray-800"></div>
					<div class="mt-3 h-3 w-56 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
					<div class="mt-2 h-3 w-full max-w-sm animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div>
				</div>
				<div class="h-9 w-28 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>
			</div>
		</div>
	{:else if error || !project}
		<div class="surface overflow-hidden">
			<ErrorState title="Could not load project" message={error || 'Project not found'} on:retry={() => void loadProject()} />
		</div>
	{:else}
		<DeployControlPanel
			{project}
			{publicProjectHost}
			{publicProjectURL}
			{pendingAction}
			on:stop={handleStop}
			on:restart={handleRestart}
			on:deploy={handleDeploy}
		/>

		<div class="py-5">
			<slot />
		</div>
	{/if}
</div>
