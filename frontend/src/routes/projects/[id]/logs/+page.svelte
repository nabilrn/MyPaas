<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { LogLine } from '$types';

	type LogEntry = LogLine & {
		id: number;
	};

	const maxLines = 5000;
	const historyService = 'app';

	let logs: LogEntry[] = [];
	let loading = true;
	let reloadingHistory = false;
	let streaming = false;
	let paused = false;
	let filter = '';
	let selectedService = 'all';
	let error = '';
	let nextID = 1;
	let logViewport: HTMLDivElement | null = null;
	let source: EventSource | null = null;

	$: services = ['all', ...Array.from(new Set(logs.map((log) => log.service))).sort()];
	$: filteredLogs = logs.filter((log) => {
		const matchesService = selectedService === 'all' || log.service === selectedService;
		const query = filter.trim().toLowerCase();
		const matchesFilter = query === '' || log.line.toLowerCase().includes(query) || log.service.toLowerCase().includes(query);
		return matchesService && matchesFilter;
	});

	onMount(() => {
		void loadHistory();
		connectStream();

		return () => {
			source?.close();
			source = null;
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
		source = new EventSource(`/api/projects/${$page.params.id}/stream`, { withCredentials: true });

		source.addEventListener('open', () => {
			streaming = true;
			error = '';
		});

		source.addEventListener('log', appendStreamLog);
		source.addEventListener('deployment-log', appendStreamLog);

		source.addEventListener('error', () => {
			streaming = false;
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
	}

	function copyVisibleLogs() {
		const text = filteredLogs.map((log) => formatLine(log)).join('\n');
		void navigator.clipboard.writeText(text)
			.then(() => toast.success('Logs copied'))
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

<div class="flex h-[calc(100vh-16rem)] min-h-[32rem] flex-col gap-3">
	<div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
		<div>
			<h1 class="text-lg font-semibold tracking-tight text-gray-950 dark:text-white">Logs</h1>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
				{streaming ? 'Streaming container output' : 'Connecting to log stream'}
				<span class="ml-2 inline-flex items-center gap-1.5">
					<span class="h-1.5 w-1.5 rounded-full {streaming ? 'bg-green-500' : 'bg-amber-500'}"></span>
					{filteredLogs.length} visible
				</span>
			</p>
		</div>

		<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
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
			<button
				type="button"
				on:click={copyVisibleLogs}
				disabled={filteredLogs.length === 0}
				class="h-9 rounded-md border border-gray-300 bg-white px-3 text-sm font-medium text-gray-800 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-900"
			>
				Copy
			</button>
			<button
				type="button"
				on:click={downloadLogs}
				disabled={filteredLogs.length === 0}
				class="h-9 rounded-md border border-gray-300 bg-white px-3 text-sm font-medium text-gray-800 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-900"
			>
				Download
			</button>
		</div>
	</div>

	{#if error}
		<div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
			{error}
		</div>
	{/if}

	<div
		bind:this={logViewport}
		on:scroll={handleScroll}
		class="scrollbar-thin relative flex-1 overflow-auto rounded-lg border border-gray-800 bg-gray-950 p-4 font-mono text-xs leading-5 text-gray-100 shadow-sm"
	>
		{#if loading}
			<p class="text-gray-500">Loading logs...</p>
		{:else if filteredLogs.length === 0}
			<p class="text-gray-500">{logs.length === 0 ? 'No logs yet.' : 'No logs match the current filter.'}</p>
		{:else}
			{#each filteredLogs as log (log.id)}
				<div class="grid grid-cols-[5.5rem_7rem_minmax(0,1fr)] gap-2 whitespace-pre-wrap break-words">
					<span class="text-gray-500">{log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : '--:--:--'}</span>
					<span class="truncate text-sky-300">{log.service}</span>
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
				<button type="button" on:click={scrollToBottom} class="font-medium text-gray-700 hover:text-gray-950 dark:text-gray-300 dark:hover:text-white">
					Resume auto-scroll
				</button>
			{/if}
			<button type="button" on:click={clearLogs} class="text-gray-500 hover:text-gray-800 dark:hover:text-gray-200">
				Clear local view
			</button>
			<ActionButton
				variant="ghost"
				size="xs"
				type="button"
				on:click={() => loadHistory(true)}
				loading={reloadingHistory}
				loadingLabel="Reloading..."
				className="min-h-0 px-0 py-0"
			>
				Reload history
			</ActionButton>
		</div>
	</div>
</div>
