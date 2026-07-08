<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import IconButton from '$components/IconButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { LogLine } from '$types';

	type LogEntry = LogLine & {
		id: number;
	};

	const maxLines = 5000;
	const renderLimit = 1000;
	const historyService = 'app';

	let logs: LogEntry[] = [];
	let loading = true;
	let reloadingHistory = false;
	let streaming = false;
	let paused = false;
	let filter = '';
	let selectedService = 'all';
	let error = '';
	let streamError = '';
	let nextID = 1;
	let logViewport: HTMLDivElement | null = null;
	let source: EventSource | null = null;
	let logsCopied = false;
	let copyResetTimer: ReturnType<typeof setTimeout> | undefined;

	$: services = ['all', ...Array.from(new Set(logs.map((log) => log.service))).sort()];
	$: filteredLogs = logs.filter((log) => {
		const matchesService = selectedService === 'all' || log.service === selectedService;
		const query = filter.trim().toLowerCase();
		const matchesFilter = query === '' || log.line.toLowerCase().includes(query) || log.service.toLowerCase().includes(query);
		return matchesService && matchesFilter;
	});
	$: renderedLogs = filteredLogs.length > renderLimit ? filteredLogs.slice(-renderLimit) : filteredLogs;
	$: clippedRenderCount = filteredLogs.length - renderedLogs.length;
	$: streamDescription = streaming
		? 'Streaming container output with local filter and copy/export controls.'
		: streamError
			? 'Live stream is reconnecting. Historical logs remain available.'
			: 'Connecting to the project log stream.';

	onMount(() => {
		void loadHistory();
		connectStream();

		return () => {
			source?.close();
			source = null;
			if (copyResetTimer) {
				clearTimeout(copyResetTimer);
			}
		};
	});

	async function loadHistory(background = false) {
		if (background) {
			reloadingHistory = true;
		} else {
			loading = true;
		}
		error = '';
		try {
			const history = await api.logs.list($page.params.id, 500);
			const now = new Date().toISOString();
			const entries = history.items?.length
				? history.items
				: history.lines.map((line) => ({ service: historyService, line }));
			logs = entries.map((item) => ({
				id: nextID++,
				service: item.service || historyService,
				line: item.line,
				timestamp: now
			}));
			await scrollToBottom();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load logs';
		} finally {
			if (background) {
				reloadingHistory = false;
			} else {
				loading = false;
			}
		}
	}

	function connectStream() {
		source?.close();
		streaming = false;
		streamError = '';
		source = new EventSource(`/api/projects/${$page.params.id}/stream`, { withCredentials: true });

		source.addEventListener('open', () => {
			streaming = true;
			streamError = '';
		});

		source.addEventListener('log', appendStreamLog);
		source.addEventListener('deployment-log', appendStreamLog);

		source.addEventListener('error', () => {
			streaming = false;
			streamError = 'Live stream disconnected. Browser retry is active.';
		});
	}

	function appendStreamLog(event: MessageEvent) {
		try {
			const parsed = JSON.parse(event.data) as Partial<LogLine>;
			if (!parsed.line) return;
			appendLog({
				id: nextID++,
				service: parsed.service || historyService,
				line: parsed.line,
				timestamp: parsed.timestamp || new Date().toISOString()
			});
		} catch {
			appendLog({
				id: nextID++,
				service: historyService,
				line: event.data,
				timestamp: new Date().toISOString()
			});
		}
	}

	function appendLog(entry: LogEntry) {
		const shouldFollow = !paused && isNearBottom();
		logs = [...logs, entry].slice(-maxLines);
		if (shouldFollow) {
			void scrollToBottom();
		}
	}

	function handleScroll() {
		paused = !isNearBottom();
	}

	function isNearBottom() {
		if (!logViewport) return true;
		const remaining = logViewport.scrollHeight - logViewport.scrollTop - logViewport.clientHeight;
		return remaining < 48;
	}

	async function scrollToBottom() {
		await tick();
		if (!logViewport) return;
		logViewport.scrollTop = logViewport.scrollHeight;
		paused = false;
	}

	function clearLogs() {
		logs = [];
		selectedService = 'all';
	}

	function reconnectStream() {
		connectStream();
	}

	function copyVisibleLogs() {
		const text = filteredLogs.map((log) => formatLine(log)).join('\n');
		void navigator.clipboard.writeText(text)
			.then(() => {
				logsCopied = true;
				if (copyResetTimer) {
					clearTimeout(copyResetTimer);
				}
				copyResetTimer = setTimeout(() => {
					logsCopied = false;
					copyResetTimer = undefined;
				}, 1800);
				toast.success('Logs copied');
			})
			.catch(() => toast.error('Failed to copy logs'));
	}

	function downloadLogs() {
		const text = filteredLogs.map((log) => formatLine(log)).join('\n');
		const blob = new Blob([text], { type: 'text/plain;charset=utf-8' });
		const url = URL.createObjectURL(blob);
		const link = document.createElement('a');
		link.href = url;
		link.download = `mypaas-${$page.params.id}-logs.txt`;
		link.click();
		URL.revokeObjectURL(url);
	}

	function formatLine(log: LogEntry) {
		const time = log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : '--:--:--';
		return `[${time}] [${log.service}] ${log.line}`;
	}
</script>

<svelte:head>
	<title>Logs · MyPaas</title>
</svelte:head>

<div class="flex h-[calc(100vh-16rem)] min-h-[32rem] flex-col">
	<SectionPanel
		title="Log stream"
		description={streamDescription}
		className="flex min-h-0 flex-1 flex-col"
		contentClass="flex min-h-0 flex-1 flex-col gap-3 p-4"
	>
		<svelte:fragment slot="actions">
			<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
				<span class="inline-flex min-h-9 items-center gap-1.5 rounded-md border border-gray-200 bg-gray-50 px-2.5 text-xs font-medium text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300">
					<span class="h-1.5 w-1.5 rounded-full {streaming ? 'bg-green-500' : 'bg-amber-500'}"></span>
					{filteredLogs.length} visible
				</span>
			<input
				type="search"
				bind:value={filter}
				placeholder="Filter logs"
				class="field h-9 w-full sm:w-56"
			/>
			<select
				bind:value={selectedService}
				class="field h-9"
			>
				{#each services as service}
					<option value={service}>{service === 'all' ? 'All services' : service}</option>
				{/each}
			</select>
			<IconButton
				label={logsCopied ? 'Logs copied' : 'Copy visible logs'}
				variant={logsCopied ? 'brand' : 'default'}
				on:click={copyVisibleLogs}
				disabled={filteredLogs.length === 0}
			>
				{#if logsCopied}
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
					</svg>
				{:else}
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M8 7h10a2 2 0 012 2v10a2 2 0 01-2 2H8a2 2 0 01-2-2V9a2 2 0 012-2z" />
						<path stroke-linecap="round" stroke-linejoin="round" d="M4 15H3a2 2 0 01-2-2V5a2 2 0 012-2h10a2 2 0 012 2v1" />
					</svg>
				{/if}
			</IconButton>
			<IconButton
				label="Download visible logs"
				variant="default"
				on:click={downloadLogs}
				disabled={filteredLogs.length === 0}
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 3v11m0 0l-4-4m4 4l4-4" />
					<path stroke-linecap="round" stroke-linejoin="round" d="M5 19h14" />
				</svg>
			</IconButton>
			</div>
		</svelte:fragment>

	{#if error}
		<div class="flex flex-col gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200 sm:flex-row sm:items-center sm:justify-between">
			<span>{error}</span>
			<ActionButton
				variant="ghost"
				size="xs"
				type="button"
				on:click={() => loadHistory(true)}
				loading={reloadingHistory}
				loadingLabel="Retrying..."
				className="text-amber-800 hover:bg-amber-100 dark:text-amber-100 dark:hover:bg-amber-900/40"
			>
				Retry history
			</ActionButton>
		</div>
	{/if}

	{#if streamError}
		<div class="flex flex-col gap-2 rounded-md border border-gray-200 bg-white px-3 py-2 text-sm text-gray-600 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300 sm:flex-row sm:items-center sm:justify-between">
			<span>{streamError}</span>
			<ActionButton variant="secondary" size="xs" type="button" on:click={reconnectStream}>
				Reconnect
			</ActionButton>
		</div>
	{/if}

	<div
		bind:this={logViewport}
		on:scroll={handleScroll}
		class="scrollbar-thin relative flex-1 overflow-auto rounded-md border border-gray-800 bg-gray-950 p-4 font-mono text-xs leading-5 text-gray-100 shadow-sm"
		aria-live="polite"
	>
		{#if loading}
			<div class="space-y-2">
				{#each [1, 2, 3, 4, 5, 6] as _}
					<div class="grid grid-cols-[4.5rem_minmax(0,1fr)] gap-2 sm:grid-cols-[5.5rem_7rem_minmax(0,1fr)]">
						<span class="h-4 animate-pulse rounded bg-gray-800"></span>
						<span class="h-4 animate-pulse rounded bg-gray-800"></span>
						<span class="h-4 animate-pulse rounded bg-gray-800"></span>
					</div>
				{/each}
			</div>
		{:else if filteredLogs.length === 0}
			<p class="text-gray-500">{logs.length === 0 ? 'No logs yet.' : 'No logs match the current filter.'}</p>
		{:else}
			{#if clippedRenderCount > 0}
				<p class="mb-2 text-gray-500">
					Rendering latest {renderLimit} of {filteredLogs.length} matching lines. Copy/download still includes all matches.
				</p>
			{/if}
			{#each renderedLogs as log (log.id)}
				<div class="grid grid-cols-[4.5rem_minmax(0,1fr)] gap-2 whitespace-pre-wrap break-words sm:grid-cols-[5.5rem_7rem_minmax(0,1fr)]">
					<span class="text-gray-500">{log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : '--:--:--'}</span>
					<span class="truncate text-sky-300 max-sm:col-start-2 max-sm:row-start-2">{log.service}</span>
					<span>{log.line}</span>
				</div>
			{/each}
		{/if}
	</div>

	<div class="flex flex-wrap items-center justify-between gap-3 text-xs text-gray-500 dark:text-gray-400">
		<div>
			Showing {filteredLogs.length} of {logs.length} lines. Keeping latest {maxLines} lines in memory.
		</div>
		<div class="flex items-center gap-3">
			{#if paused}
				<IconButton label="Resume auto-scroll" variant="brand" type="button" on:click={scrollToBottom}>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14m0 0l-5-5m5 5l5-5" />
					</svg>
				</IconButton>
			{/if}
			<IconButton label="Clear local log view" variant="ghost" type="button" on:click={clearLogs} disabled={logs.length === 0}>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M4 7h16" />
					<path stroke-linecap="round" stroke-linejoin="round" d="M10 11v6M14 11v6M6 7l1 14h10l1-14M9 7V4h6v3" />
				</svg>
			</IconButton>
			<IconButton
				label="Reload log history"
				variant="brand"
				type="button"
				on:click={() => loadHistory(true)}
				loading={reloadingHistory}
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M20 11a8.1 8.1 0 00-15.5-3M4 4v4h4m-4 5a8.1 8.1 0 0015.5 3M20 20v-4h-4" />
				</svg>
			</IconButton>
		</div>
	</div>
	</SectionPanel>
</div>
