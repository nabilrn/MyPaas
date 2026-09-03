<script context="module" lang="ts">
	type DatabaseShortcutState = {
		configured: boolean;
		driver: string;
		database: string;
		available: boolean;
	};

	const cache = new Map<string, DatabaseShortcutState>();
	const inFlight = new Map<string, Promise<DatabaseShortcutState>>();

	async function fetchConfiguration(projectId: string): Promise<DatabaseShortcutState> {
		const cached = cache.get(projectId);
		if (cached) return cached;
		const existing = inFlight.get(projectId);
		if (existing) return existing;

		const request = (async () => {
			try {
				const response = await fetch(`/api/projects/${encodeURIComponent(projectId)}/db/status?probe=false`, {
					credentials: 'include',
					headers: { Accept: 'application/json' }
				});
				if (!response.ok) {
					return { configured: false, driver: '', database: '', available: false };
				}
				const body = (await response.json()) as {
					data?: {
						configured?: boolean;
						connection?: { driver?: string; database?: string } | null;
					};
				};
				const status = body.data;
				const result = {
					configured: Boolean(status?.configured),
					driver: status?.connection?.driver ?? '',
					database: status?.connection?.database ?? '',
					available: true
				};
				cache.set(projectId, result);
				return result;
			} catch {
				return { configured: false, driver: '', database: '', available: false };
			} finally {
				inFlight.delete(projectId);
			}
		})();

		inFlight.set(projectId, request);
		return request;
	}
</script>

<script lang="ts">
	import { Database } from '@lucide/svelte';

	export let projectId: string;

	let loadedProjectId = '';
	let loading = true;
	let state: DatabaseShortcutState | null = null;

	$: if (projectId && projectId !== loadedProjectId) {
		loadedProjectId = projectId;
		void load(projectId);
	}

	async function load(id: string) {
		loading = true;
		state = await fetchConfiguration(id);
		if (loadedProjectId === id) loading = false;
	}

	function driverLabel(driver: string) {
		if (driver === 'postgres') return 'Postgres';
		if (driver === 'mysql') return 'MySQL';
		if (driver === 'mariadb') return 'MariaDB';
		if (driver === 'sqlite') return 'SQLite';
		return 'Database';
	}
</script>

{#if loading}
	<span class="text-sm text-gray-400 dark:text-gray-500" aria-label="Checking database configuration">…</span>
{:else if state?.available && state.configured}
	<a
		href={`/projects/${projectId}/database`}
		class="app-focus inline-flex max-w-full items-center gap-1.5 rounded-sm text-sm font-medium text-gray-700 hover:text-gray-950 hover:underline dark:text-gray-300 dark:hover:text-white"
		title={state.database ? `Open ${driverLabel(state.driver)} database ${state.database}` : 'Open Database Studio'}
	>
		<Database class="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
		<span class="truncate">{driverLabel(state.driver)}</span>
	</a>
{:else if state?.available}
	<span class="whitespace-nowrap text-sm text-gray-400 dark:text-gray-500">Not configured</span>
{:else}
	<span class="text-sm text-gray-400 dark:text-gray-500">—</span>
{/if}
