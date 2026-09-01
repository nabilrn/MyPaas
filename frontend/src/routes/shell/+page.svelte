<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import { CircleStop, RefreshCw, Terminal, TriangleAlert } from '@lucide/svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { ShellSession } from '$types';

	type ShellStatus = 'idle' | 'connecting' | 'connected' | 'disconnected' | 'ended';

	const MAX_OUTPUT_CHARS = 200_000;
	let session: ShellSession | null = null;
	let stream: EventSource | null = null;
	let output = '';
	let command = '';
	let error = '';
	let status: ShellStatus = 'idle';
	let starting = false;
	let sending = false;
	let stopping = false;
	let outputElement: HTMLPreElement | null = null;
	let commandInput: HTMLInputElement | null = null;
	let mounted = false;

	$: statusLabel = {
		idle: 'Not started',
		connecting: 'Connecting',
		connected: 'Connected',
		disconnected: 'Disconnected',
		ended: 'Ended'
	}[status];
	$: statusClass = {
		idle: 'bg-gray-400',
		connecting: 'bg-amber-500',
		connected: 'bg-emerald-500',
		disconnected: 'bg-red-500',
		ended: 'bg-gray-400'
	}[status];

	onMount(() => {
		mounted = true;
		void startSession();
		return () => {
			mounted = false;
			closeStream();
			if (session) void api.admin.shell.stopSession(session.id);
		};
	});

	onDestroy(closeStream);

	async function startSession() {
		if (starting || session) return;
		starting = true;
		error = '';
		output = '';
		status = 'connecting';
		try {
			const nextSession = await api.admin.shell.startSession();
			if (!mounted) {
				void api.admin.shell.stopSession(nextSession.id);
				return;
			}
			session = nextSession;
			connectStream(nextSession.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to start host shell';
			status = 'ended';
		} finally {
			starting = false;
		}
	}

	function connectStream(id: string) {
		closeStream();
		const nextStream = new EventSource(`/api/admin/shell/sessions/${id}/stream`);
		stream = nextStream;
		nextStream.addEventListener('ready', () => {
			if (mounted) {
				status = 'connected';
				void focusCommandInput();
			}
		});
		nextStream.addEventListener('output', (event) => {
			const payload = (event as MessageEvent<string>).data;
			try {
				appendOutput(JSON.parse(payload) as string);
			} catch {
				appendOutput(payload);
			}
		});
		nextStream.addEventListener('exit', (event) => {
			appendOutput(`\n${(event as MessageEvent<string>).data}\n`);
			status = 'ended';
			closeStream();
			session = null;
		});
		nextStream.onerror = () => {
			if (mounted && status !== 'ended') status = 'disconnected';
		};
	}

	function closeStream() {
		stream?.close();
		stream = null;
	}

	function appendOutput(chunk: string) {
		output = `${output}${chunk}`.slice(-MAX_OUTPUT_CHARS);
		requestAnimationFrame(() => {
			if (outputElement) outputElement.scrollTop = outputElement.scrollHeight;
		});
	}

	async function sendCommand() {
		const value = command;
		if (!session || !value.trim() || sending || status !== 'connected') return;
		sending = true;
		error = '';
		try {
			await api.admin.shell.sendInput(session.id, `${value}\n`);
			command = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to send shell input';
		} finally {
			sending = false;
			await focusCommandInput();
		}
	}

	async function focusCommandInput() {
		await tick();
		commandInput?.focus();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			void sendCommand();
		}
	}

	async function stopSession() {
		if (!session || stopping) return;
		stopping = true;
		try {
			await api.admin.shell.stopSession(session.id);
			toast.success('Shell session ended');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to end shell session');
		} finally {
			closeStream();
			session = null;
			status = 'ended';
			stopping = false;
		}
	}

	function reconnect() {
		if (session) connectStream(session.id);
		else void startSession();
	}
</script>

<svelte:head>
	<title>Shell · MyPaaS</title>
</svelte:head>

<div class="flex h-[calc(100vh-3.5rem)] min-h-[30rem] flex-col overflow-hidden bg-white dark:bg-neutral-950">
	<div class="flex min-h-12 shrink-0 flex-wrap items-center justify-between gap-2 border-b border-gray-200 px-4 py-2 dark:border-neutral-800 lg:px-5">
		<div class="flex min-w-0 items-center gap-3">
			<span class="inline-flex items-center gap-2 text-sm font-medium text-gray-800 dark:text-gray-200">
				<span class={`status-dot ${statusClass}`}></span>
				{statusLabel}
			</span>
			{#if session}
				<span class="hidden text-[13px] text-gray-500 dark:text-gray-400 sm:inline">Session expires {new Date(session.expiresAt).toLocaleTimeString()}</span>
			{/if}
		</div>

		<div class="flex min-w-0 items-center gap-2">
			<span class="hidden items-center gap-1.5 text-[13px] text-gray-500 dark:text-gray-400 xl:inline-flex">
				<TriangleAlert class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
				Commands affect the entire host
			</span>
			{#if session && status === 'disconnected'}
				<ActionButton variant="secondary" size="xs" on:click={reconnect}>
					<RefreshCw slot="icon" class="h-3.5 w-3.5" />
					Reconnect
				</ActionButton>
			{/if}
			{#if session}
				<ActionButton variant="ghostDanger" size="xs" loading={stopping} loadingLabel="Ending" on:click={stopSession}>
					<CircleStop slot="icon" class="h-3.5 w-3.5" />
					End session
				</ActionButton>
			{/if}
		</div>
	</div>

	{#if error}
		<div class="shrink-0 border-b border-red-200 bg-red-50 px-4 py-2 text-[13px] text-red-700 dark:border-red-950 dark:bg-red-950/30 dark:text-red-300" role="alert">{error}</div>
	{/if}

	<div class="min-h-0 flex-1 bg-neutral-900 dark:bg-neutral-950">
		{#if session}
			<pre
				bind:this={outputElement}
				class="h-full overflow-auto whitespace-pre-wrap break-words bg-neutral-900 p-4 font-mono text-sm leading-6 text-gray-200 dark:bg-neutral-950"
				aria-label="Shell output"
			>{output || 'Waiting for shell output…'}</pre>
		{:else}
			<div class="flex h-full items-center justify-center bg-neutral-900 p-6 dark:bg-neutral-950">
				<div class="flex flex-col items-center gap-3 text-center">
					<Terminal class="h-6 w-6 text-gray-600" aria-hidden="true" />
					<p class="text-sm text-gray-400">{status === 'connecting' ? 'Opening host shell…' : 'No shell session is running.'}</p>
					{#if status !== 'connecting'}
						<ActionButton variant="secondary" size="sm" on:click={startSession} loading={starting} loadingLabel="Starting">
							<Terminal slot="icon" class="h-4 w-4" />
							Start shell
						</ActionButton>
					{/if}
				</div>
			</div>
		{/if}
	</div>

	{#if session}
		<form class="flex h-12 shrink-0 items-center gap-2 border-t border-neutral-700 bg-neutral-900 px-4 dark:border-neutral-800 dark:bg-neutral-950" on:submit|preventDefault={sendCommand}>
			<label class="font-mono text-sm text-gray-500" for="shell-command">$</label>
			<input
				bind:this={commandInput}
				id="shell-command"
				class="min-w-0 flex-1 border-0 bg-transparent px-0 font-mono text-sm text-gray-100 outline-none placeholder:text-gray-600 focus:ring-0"
				bind:value={command}
				on:keydown={handleKeydown}
				placeholder={status === 'connected' ? 'Enter a command' : 'Waiting for shell connection'}
				disabled={status !== 'connected' || sending}
				autocomplete="off"
			/>
			<ActionButton type="submit" variant="secondary" size="xs" loading={sending} loadingLabel="Sending" disabled={status !== 'connected' || !command.trim()}>
				Send
			</ActionButton>
		</form>
	{/if}
</div>
