<script lang="ts">
	import { ArrowDown, Check, Copy, Download, RefreshCw, Trash2 } from '@lucide/svelte';
	import { onMount, tick } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import { reconnectProjectStream, projectStreamConnection, projectStreamLogs } from '$stores/project-stream';
	import { toast } from '$stores/toast';
	import type { LogLine } from '$types';

	type LogEntry = LogLine & { id: number };
	const maxLines = 5000;
	const renderLimit = 1000;
	const historyService = 'app';

	let logs: LogEntry[] = [];
	let loading = true;
	let reloadingHistory = false;
	let paused = false;
	let filter = '';
	let selectedService = 'all';
	let error = '';
	let nextID = 1;
	let consumedStreamLogs = 0;
	let logViewport: HTMLDivElement | null = null;
	let logsCopied = false;
	let copyResetTimer: ReturnType<typeof setTimeout> | undefined;

	$: streaming = $projectStreamConnection === 'open';
	$: streamError = $projectStreamConnection === 'reconnecting' ? 'Live stream disconnected. Browser retry is active.' : '';
	$: streamDescription = streaming ? 'Streaming container output with local filtering and export controls.' : streamError ? 'Live stream is reconnecting. Historical logs remain available.' : 'Connecting to the project log stream.';
	$: if ($projectStreamLogs.length < consumedStreamLogs) consumedStreamLogs = 0;
	$: if ($projectStreamLogs.length > consumedStreamLogs) {
		const incoming = $projectStreamLogs.slice(consumedStreamLogs);
		consumedStreamLogs = $projectStreamLogs.length;
		for (const item of incoming) appendLog({ id: nextID++, ...item });
	}
	$: services = ['all', ...Array.from(new Set(logs.map((log) => log.service))).sort()];
	$: filteredLogs = logs.filter((log) => {
		const matchesService = selectedService === 'all' || log.service === selectedService;
		const query = filter.trim().toLowerCase();
		return matchesService && (query === '' || log.line.toLowerCase().includes(query) || log.service.toLowerCase().includes(query));
	});
	$: renderedLogs = filteredLogs.length > renderLimit ? filteredLogs.slice(-renderLimit) : filteredLogs;
	$: clippedRenderCount = filteredLogs.length - renderedLogs.length;

	onMount(() => {
		void loadHistory();
		return () => { if (copyResetTimer) clearTimeout(copyResetTimer); };
	});

	async function loadHistory(background = false) {
		if (background) reloadingHistory = true;
		else loading = true;
		error = '';
		try {
			const history = await api.logs.list($page.params.id ?? '', 500);
			const now = new Date().toISOString();
			const entries = history.items?.length ? history.items : history.lines.map((line) => ({ service: historyService, line }));
			logs = entries.map((item) => ({ id: nextID++, service: item.service || historyService, line: item.line, timestamp: now }));
			await scrollToBottom();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load logs';
		} finally {
			if (background) reloadingHistory = false;
			else loading = false;
		}
	}

	function appendLog(entry: LogEntry) {
		const shouldFollow = !paused && isNearBottom();
		logs = [...logs, entry].slice(-maxLines);
		if (shouldFollow) void scrollToBottom();
	}
	function handleScroll() { paused = !isNearBottom(); }
	function isNearBottom() { if (!logViewport) return true; return logViewport.scrollHeight - logViewport.scrollTop - logViewport.clientHeight < 48; }
	async function scrollToBottom() { await tick(); if (!logViewport) return; logViewport.scrollTop = logViewport.scrollHeight; paused = false; }
	function clearLogs() { logs = []; selectedService = 'all'; }
	function copyVisibleLogs() {
		void navigator.clipboard.writeText(filteredLogs.map(formatLine).join('\n')).then(() => {
			logsCopied = true;
			if (copyResetTimer) clearTimeout(copyResetTimer);
			copyResetTimer = setTimeout(() => { logsCopied = false; copyResetTimer = undefined; }, 1800);
			toast.success('Logs copied');
		}).catch(() => toast.error('Failed to copy logs'));
	}
	function downloadLogs() {
		const blob = new Blob([filteredLogs.map(formatLine).join('\n')], { type: 'text/plain;charset=utf-8' });
		const url = URL.createObjectURL(blob);
		const link = document.createElement('a'); link.href = url; link.download = `mypaas-${$page.params.id}-logs.txt`; link.click(); URL.revokeObjectURL(url);
	}
	function formatLine(log: LogEntry) { const time = log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : '--:--:--'; return `[${time}] [${log.service}] ${log.line}`; }
</script>

<svelte:head><title>Logs · MyPaas</title></svelte:head>

<div class="flex h-[calc(100vh-16rem)] min-h-[32rem] flex-col">
	<SectionPanel title="Log stream" description={streamDescription} className="flex min-h-0 flex-1 flex-col" contentClass="flex min-h-0 flex-1 flex-col gap-3 p-4">
		<svelte:fragment slot="actions">
			<div class="flex flex-wrap items-center gap-2">
				<span class="inline-flex h-9 items-center gap-2 px-1 text-sm text-gray-500 dark:text-gray-400"><span class="status-dot {streaming ? 'bg-emerald-500' : 'bg-amber-500'}"></span>{filteredLogs.length} visible</span>
				<input type="search" bind:value={filter} placeholder="Filter logs" class="field h-9 w-full sm:w-56" />
				<select bind:value={selectedService} class="field h-9 min-w-36">{#each services as service}<option value={service}>{service === 'all' ? 'All services' : service}</option>{/each}</select>
				<ActionButton variant="secondary" size="sm" on:click={copyVisibleLogs} disabled={filteredLogs.length === 0}>{#if logsCopied}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}{logsCopied ? 'Copied' : 'Copy'}</ActionButton>
				<ActionButton variant="secondary" size="sm" on:click={downloadLogs} disabled={filteredLogs.length === 0}><Download slot="icon" class="h-4 w-4" />Download</ActionButton>
			</div>
		</svelte:fragment>
		{#if error}<div class="alert-warning flex-wrap items-center justify-between"><span class="min-w-0 flex-1">{error}</span><ActionButton variant="ghost" size="xs" type="button" on:click={() => loadHistory(true)} loading={reloadingHistory} loadingLabel="Retrying"><RefreshCw slot="icon" class="h-3.5 w-3.5" />Retry history</ActionButton></div>{/if}
		{#if streamError}<div class="alert-neutral flex-wrap items-center justify-between"><span class="min-w-0 flex-1">{streamError}</span><ActionButton variant="secondary" size="xs" type="button" on:click={reconnectProjectStream}><RefreshCw slot="icon" class="h-3.5 w-3.5" />Reconnect</ActionButton></div>{/if}
		<div bind:this={logViewport} on:scroll={handleScroll} class="console-surface scrollbar-thin relative flex-1 overflow-auto p-4" aria-live="polite">
			{#if loading}<div class="space-y-2">{#each [1,2,3,4,5,6] as _}<div class="grid grid-cols-[4.5rem_minmax(0,1fr)] gap-2 sm:grid-cols-[5.5rem_7rem_minmax(0,1fr)]"><span class="h-4 animate-pulse rounded bg-gray-800"></span><span class="h-4 animate-pulse rounded bg-gray-800"></span><span class="h-4 animate-pulse rounded bg-gray-800"></span></div>{/each}</div>
			{:else if filteredLogs.length === 0}<p class="text-gray-500">{logs.length === 0 ? 'No logs yet.' : 'No logs match the current filter.'}</p>
			{:else}{#if clippedRenderCount > 0}<p class="mb-2 text-gray-500">Rendering latest {renderLimit} of {filteredLogs.length} matching lines. Copy/download still includes all matches.</p>{/if}{#each renderedLogs as log (log.id)}<div class="grid grid-cols-[4.5rem_minmax(0,1fr)] gap-2 whitespace-pre-wrap break-words sm:grid-cols-[5.5rem_7rem_minmax(0,1fr)]"><span class="text-gray-500">{log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : '--:--:--'}</span><span class="truncate text-sky-300 max-sm:col-start-2 max-sm:row-start-2">{log.service}</span><span>{log.line}</span></div>{/each}{/if}
		</div>
		<div class="flex flex-wrap items-center justify-between gap-3 text-sm text-gray-500 dark:text-gray-400"><p>Showing {filteredLogs.length} of {logs.length} lines · latest {maxLines} kept in memory.</p><div class="flex flex-wrap items-center gap-2">{#if paused}<ActionButton variant="secondary" size="xs" type="button" on:click={scrollToBottom}><ArrowDown slot="icon" class="h-3.5 w-3.5" />Resume</ActionButton>{/if}<ActionButton variant="ghost" size="xs" type="button" on:click={clearLogs} disabled={logs.length === 0}><Trash2 slot="icon" class="h-3.5 w-3.5" />Clear</ActionButton><ActionButton variant="secondary" size="xs" type="button" on:click={() => loadHistory(true)} loading={reloadingHistory} loadingLabel="Reloading"><RefreshCw slot="icon" class="h-3.5 w-3.5" />Reload history</ActionButton></div></div>
	</SectionPanel>
</div>
