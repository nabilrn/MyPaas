<script lang="ts">
	import { onMount } from 'svelte';
	import { LoaderCircle } from '@lucide/svelte';
	import { api, type HostStats } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';

	type SettingKey = 'user_ram_quota_gb' | 'user_cpu_quota' | 'max_projects' | 'build_timeout_minutes';
	type NumericSettings = Record<SettingKey, number>;
	type UpdateState = 'available' | 'current' | 'unavailable' | 'unknown' | 'tracking_ref';

	interface UpdateStatus {
		state: UpdateState;
		channel: string;
		current_build_sha?: string;
		current_tag?: string;
		latest_tag?: string;
		latest_sha?: string;
		release_url?: string;
		published_at?: string;
		tracking_ref?: string;
		update_available: boolean;
	}

	const defaultSettings: NumericSettings = {
		user_ram_quota_gb: 0,
		user_cpu_quota: 0,
		max_projects: 0,
		build_timeout_minutes: 0
	};

	let settings: NumericSettings = { ...defaultSettings };
	let savedSettings: NumericSettings = { ...defaultSettings };
	let hostStats: HostStats | null = null;
	let loadingSettings = true;
	let savingSettings = false;
	let triggeringUpdate = false;
	let updateOverlayOpen = false;
	let currentBuildSha = '';
	let updateStatus: UpdateStatus | null = null;

	$: settingsChanged = (Object.keys(defaultSettings) as SettingKey[]).some((key) => settings[key] !== savedSettings[key]);
	$: validationErrors = {
		user_ram_quota_gb: numberError(settings.user_ram_quota_gb, 0, 64, false, 'RAM quota must be greater than 0 and at most 64 GB.'),
		user_cpu_quota: numberError(settings.user_cpu_quota, 0, 32, false, 'CPU quota must be greater than 0 and at most 32 cores.'),
		max_projects: numberError(settings.max_projects, 1, 500, true, 'Maximum projects must be a whole number between 1 and 500.'),
		build_timeout_minutes: numberError(settings.build_timeout_minutes, 1, 1440, true, 'Build timeout must be a whole number between 1 and 1440 minutes.')
	};
	$: hasValidationErrors = Object.values(validationErrors).some(Boolean);
	$: hostMemoryTotal = hostStats?.memory?.total_bytes ?? hostStats?.host_ram_bytes ?? 0;
	$: hostMemoryUsed = hostStats?.memory ? Math.max(0, hostStats.memory.total_bytes - hostStats.memory.available_bytes) : 0;
	$: hostStorageUsed = hostStats?.storage ? Math.max(0, hostStats.storage.total_bytes - hostStats.storage.available_bytes) : 0;

	onMount(() => {
		void loadSettings();
	});

	async function loadSettings() {
		loadingSettings = true;
		try {
			const [data, capacity] = await Promise.all([
				api.admin.getSettings(),
				api.admin.getHostStats().catch(() => null)
			]);
			settings = {
				user_ram_quota_gb: numericValue(data.user_ram_quota_gb),
				user_cpu_quota: numericValue(data.user_cpu_quota),
				max_projects: numericValue(data.max_projects),
				build_timeout_minutes: numericValue(data.build_timeout_minutes)
			};
			const raw = data as unknown as Record<string, unknown>;
			currentBuildSha = stringValue(raw.build_sha);
			updateStatus = parseUpdateStatus(raw.update_status);
			savedSettings = { ...settings };
			hostStats = capacity;
		} catch (error) {
			toast.error('Failed to load settings');
			console.error(error);
		} finally {
			loadingSettings = false;
		}
	}

	async function saveSettings() {
		if (savingSettings || !settingsChanged || hasValidationErrors) return;
		savingSettings = true;
		try {
			const updated = await api.admin.updateSettings(settings);
			settings = {
				user_ram_quota_gb: numericValue(updated.user_ram_quota_gb, settings.user_ram_quota_gb),
				user_cpu_quota: numericValue(updated.user_cpu_quota, settings.user_cpu_quota),
				max_projects: numericValue(updated.max_projects, settings.max_projects),
				build_timeout_minutes: numericValue(updated.build_timeout_minutes, settings.build_timeout_minutes)
			};
			savedSettings = { ...settings };
			toast.success('Platform settings saved');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to save settings');
			console.error(error);
		} finally {
			savingSettings = false;
		}
	}

	async function triggerUpdate() {
		if (!updateStatus?.update_available || triggeringUpdate) return;
		triggeringUpdate = true;
		try {
			await api.admin.triggerUpdate();
			updateOverlayOpen = true;
			startUpdatePolling();
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to trigger update');
			console.error(error);
		} finally {
			triggeringUpdate = false;
		}
	}

	function startUpdatePolling() {
		let wasDown = false;
		const poll = setInterval(async () => {
			try {
				const res = await fetch('/api/health');
				if (res.ok) {
					if (wasDown) {
						clearInterval(poll);
						window.location.href = '/';
					}
				} else {
					wasDown = true;
				}
			} catch {
				wasDown = true;
			}
		}, 3000);
	}

	function discardChanges() {
		settings = { ...savedSettings };
	}

	function numericValue(value: unknown, fallback = 0) {
		return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
	}

	function stringValue(value: unknown) {
		return typeof value === 'string' ? value : '';
	}

	function parseUpdateStatus(value: unknown): UpdateStatus | null {
		if (!value || typeof value !== 'object' || Array.isArray(value)) return null;
		const raw = value as Record<string, unknown>;
		const state = stringValue(raw.state) as UpdateState;
		if (!['available', 'current', 'unavailable', 'unknown', 'tracking_ref'].includes(state)) return null;
		return {
			state,
			channel: stringValue(raw.channel),
			current_build_sha: stringValue(raw.current_build_sha) || undefined,
			current_tag: stringValue(raw.current_tag) || undefined,
			latest_tag: stringValue(raw.latest_tag) || undefined,
			latest_sha: stringValue(raw.latest_sha) || undefined,
			release_url: stringValue(raw.release_url) || undefined,
			published_at: stringValue(raw.published_at) || undefined,
			tracking_ref: stringValue(raw.tracking_ref) || undefined,
			update_available: raw.update_available === true
		};
	}

	function numberError(value: number, min: number, max: number, integer: boolean, message: string) {
		if (!Number.isFinite(value) || value < min || value > max || (min === 0 && value === 0) || (integer && !Number.isInteger(value))) return message;
		return '';
	}

	function formatBytes(value: number) {
		if (!Number.isFinite(value) || value <= 0) return 'Unavailable';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		let amount = value;
		let index = 0;
		while (amount >= 1024 && index < units.length - 1) {
			amount /= 1024;
			index += 1;
		}
		return `${amount.toFixed(amount >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
	}
</script>

<svelte:head>
	<title>Settings · MyPaas</title>
</svelte:head>

{#if updateOverlayOpen}
	<div class="fixed inset-0 z-50 flex flex-col items-center justify-center bg-white/90 backdrop-blur-sm dark:bg-gray-950/90">
		<LoaderCircle class="mb-4 h-12 w-12 animate-spin text-gray-500 dark:text-gray-400" />
		<h2 class="text-xl font-medium text-gray-900 dark:text-white">Updating MyPaas</h2>
		<p class="mt-2 text-sm text-gray-500 dark:text-gray-400">Restarting control plane…</p>
	</div>
{/if}

<div class="page-shell space-y-4 py-6">
	<SectionPanel title="Host capacity" contentClass="p-0">
		{#if hostStats}
			<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 sm:grid-cols-3 sm:divide-x sm:divide-y-0">
				<div class="p-4">
					<p class="metric-label">Memory</p>
					<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{formatBytes(hostMemoryTotal)}</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{formatBytes(hostStats.allocated_ram_mb * 1024 * 1024)} allocated{hostStats.memory ? ` · ${formatBytes(hostMemoryUsed)} used` : ''}</p>
				</div>
				<div class="p-4">
					<p class="metric-label">CPU</p>
					<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{hostStats.host_cpu_cores} core{hostStats.host_cpu_cores === 1 ? '' : 's'}</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{hostStats.allocated_cpu.toFixed(2)} allocated</p>
				</div>
				<div class="p-4">
					<p class="metric-label">Storage</p>
					<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{hostStats.storage ? formatBytes(hostStats.storage.total_bytes) : 'Unavailable'}</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{hostStats.storage ? `${formatBytes(hostStorageUsed)} used · ${formatBytes(hostStats.storage.available_bytes)} free` : 'Telemetry unavailable'}</p>
				</div>
			</div>
		{:else}
			<p class="p-4 text-sm text-gray-500 dark:text-gray-400">Capacity unavailable.</p>
		{/if}
	</SectionPanel>

	{#if loadingSettings}
		<div class="surface flex h-36 items-center justify-center">
			<LoaderCircle class="h-6 w-6 animate-spin motion-reduce:animate-none text-gray-500 dark:text-gray-400" aria-hidden="true" />
		</div>
	{:else}
		<SectionPanel title="Platform limits">
			<div class="grid gap-5 lg:grid-cols-3">
				<label class="block" for="user_ram_quota_gb">
					<span class="field-label">RAM quota per user</span>
					<div class="flex items-center gap-2"><input type="number" id="user_ram_quota_gb" min="0.25" max="64" step="0.25" bind:value={settings.user_ram_quota_gb} class="field min-w-0 flex-1" aria-invalid={validationErrors.user_ram_quota_gb ? 'true' : undefined} /><span class="w-14 shrink-0 text-xs text-gray-500 dark:text-gray-400">GB</span></div>
					{#if validationErrors.user_ram_quota_gb}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.user_ram_quota_gb}</p>{/if}
				</label>
				<label class="block" for="user_cpu_quota">
					<span class="field-label">CPU quota per user</span>
					<div class="flex items-center gap-2"><input type="number" id="user_cpu_quota" min="0.1" max="32" step="0.1" bind:value={settings.user_cpu_quota} class="field min-w-0 flex-1" aria-invalid={validationErrors.user_cpu_quota ? 'true' : undefined} /><span class="w-14 shrink-0 text-xs text-gray-500 dark:text-gray-400">cores</span></div>
					{#if validationErrors.user_cpu_quota}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.user_cpu_quota}</p>{/if}
				</label>
				<label class="block" for="max_projects">
					<span class="field-label">Projects per user</span>
					<div class="flex items-center gap-2"><input type="number" id="max_projects" min="1" max="500" step="1" bind:value={settings.max_projects} class="field min-w-0 flex-1" aria-invalid={validationErrors.max_projects ? 'true' : undefined} /><span class="w-14 shrink-0 text-xs text-gray-500 dark:text-gray-400">projects</span></div>
					{#if validationErrors.max_projects}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.max_projects}</p>{/if}
				</label>
			</div>
		</SectionPanel>

		<SectionPanel title="Project defaults">
			<div class="grid gap-px overflow-hidden rounded-md border border-gray-200 bg-gray-100 dark:border-neutral-800 dark:bg-neutral-800 sm:grid-cols-2 xl:grid-cols-4">
				{#each [
					{ name: 'Static', detail: '64 MB · 0.10 CPU' },
					{ name: 'Go small', detail: '128 MB · 0.20 CPU' },
					{ name: 'Node / Python', detail: '256 MB · 0.35 CPU' },
					{ name: 'Compose main', detail: '256 MB · 0.35 CPU' }
				] as profile}
					<div class="bg-white p-3 dark:bg-neutral-950">
						<p class="text-sm font-medium text-gray-950 dark:text-white">{profile.name}</p>
						<p class="metric-value mt-1 text-xs text-gray-500 dark:text-gray-400">{profile.detail}</p>
					</div>
				{/each}
			</div>
			<p class="mt-3 text-xs text-gray-500 dark:text-gray-400">Selected during project creation; overridable per project.</p>
		</SectionPanel>

		<SectionPanel title="Deployment">
			<div class="max-w-md">
				<label class="block" for="build_timeout_minutes">
					<span class="field-label">Build timeout</span>
					<div class="flex items-center gap-2"><input type="number" id="build_timeout_minutes" min="1" max="1440" step="1" bind:value={settings.build_timeout_minutes} class="field min-w-0 flex-1" aria-invalid={validationErrors.build_timeout_minutes ? 'true' : undefined} /><span class="w-16 shrink-0 text-xs text-gray-500 dark:text-gray-400">minutes</span></div>
					{#if validationErrors.build_timeout_minutes}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.build_timeout_minutes}</p>{/if}
				</label>
			</div>
			<p class="mt-3 text-xs text-gray-500 dark:text-gray-400">Deploy concurrency is configured at process startup with <span class="font-mono">MAX_CONCURRENT_DEPLOYS</span>.</p>
		</SectionPanel>

		<SectionPanel title="System update">
			<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div class="min-w-0">
					<div class="flex flex-wrap items-center gap-2">
						{#if updateStatus?.state === 'available'}
							<span class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-950 dark:text-white"><span class="status-dot bg-amber-500"></span>Update available</span>
						{:else if updateStatus?.state === 'current'}
							<span class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-950 dark:text-white"><span class="status-dot bg-emerald-500"></span>Up to date</span>
						{:else if updateStatus?.state === 'tracking_ref'}
							<span class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-950 dark:text-white"><span class="status-dot bg-sky-500"></span>Tracking ref</span>
						{:else}
							<span class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-950 dark:text-white"><span class="status-dot bg-gray-400"></span>Release check unavailable</span>
						{/if}
					</div>
					{#if updateStatus?.state === 'available'}
						<p class="mt-1 text-xs text-gray-500 dark:text-gray-400"><span class="font-mono">{updateStatus.current_tag ?? currentBuildSha.substring(0, 12) || 'unknown'}</span> → <span class="font-mono text-gray-800 dark:text-gray-200">{updateStatus.latest_tag}</span></p>
					{:else if updateStatus?.state === 'current'}
						<p class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">{updateStatus.current_tag ?? updateStatus.latest_tag ?? currentBuildSha.substring(0, 12)}</p>
					{:else if updateStatus?.state === 'tracking_ref'}
						<p class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">{updateStatus.tracking_ref ?? 'main'}</p>
					{:else}
						<p class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">{currentBuildSha ? currentBuildSha.substring(0, 12) : 'Unknown build'}</p>
					{/if}
				</div>
				{#if updateStatus?.state === 'available'}
					<ActionButton variant="primary" size="sm" loading={triggeringUpdate} on:click={triggerUpdate}>Update to {updateStatus.latest_tag}</ActionButton>
				{:else}
					<ActionButton variant="secondary" size="sm" on:click={() => void loadSettings()} disabled={loadingSettings}>Check release</ActionButton>
				{/if}
			</div>
		</SectionPanel>

		{#if settingsChanged}
			<div class="surface flex flex-wrap items-center justify-between gap-3 px-4 py-3">
				<p class="inline-flex items-center gap-2 text-sm font-medium text-gray-950 dark:text-white"><span class="status-dot bg-amber-500"></span>Unsaved changes</p>
				<div class="flex items-center gap-2">
					<ActionButton variant="secondary" size="sm" on:click={discardChanges} disabled={savingSettings}>Discard</ActionButton>
					<ActionButton variant="primary" size="sm" loading={savingSettings} loadingLabel="Saving" on:click={saveSettings} disabled={hasValidationErrors}>Save</ActionButton>
				</div>
			</div>
		{/if}
	{/if}
</div>