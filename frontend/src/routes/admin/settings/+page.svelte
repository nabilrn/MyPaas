<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { Cpu, HardDrive, MemoryStick, RefreshCw, SlidersHorizontal, Timer } from '@lucide/svelte';
	import { api, type HostStats } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';

	type SettingKey =
		| 'build_timeout_minutes'
		| 'profile_static_memory_mb'
		| 'profile_static_cpu_limit'
		| 'profile_go_small_memory_mb'
		| 'profile_go_small_cpu_limit'
		| 'profile_node_python_memory_mb'
		| 'profile_node_python_cpu_limit'
		| 'profile_compose_main_memory_mb'
		| 'profile_compose_main_cpu_limit';
	type NumericSettings = Record<SettingKey, number>;
	type ProfileSetting = {
		name: string;
		memoryKey: Extract<SettingKey, `${string}_memory_mb`>;
		cpuKey: Extract<SettingKey, `${string}_cpu_limit`>;
		minimumMemory: number;
		minimumCPU: number;
	};

	const defaultSettings: NumericSettings = {
		build_timeout_minutes: 0,
		profile_static_memory_mb: 64,
		profile_static_cpu_limit: 0.1,
		profile_go_small_memory_mb: 128,
		profile_go_small_cpu_limit: 0.2,
		profile_node_python_memory_mb: 256,
		profile_node_python_cpu_limit: 0.35,
		profile_compose_main_memory_mb: 256,
		profile_compose_main_cpu_limit: 0.35
	};
	const profileSettings: ProfileSetting[] = [
		{ name: 'Static', memoryKey: 'profile_static_memory_mb', cpuKey: 'profile_static_cpu_limit', minimumMemory: 64, minimumCPU: 0.1 },
		{ name: 'Go small', memoryKey: 'profile_go_small_memory_mb', cpuKey: 'profile_go_small_cpu_limit', minimumMemory: 128, minimumCPU: 0.2 },
		{ name: 'Node / Python', memoryKey: 'profile_node_python_memory_mb', cpuKey: 'profile_node_python_cpu_limit', minimumMemory: 256, minimumCPU: 0.35 },
		{ name: 'Compose main', memoryKey: 'profile_compose_main_memory_mb', cpuKey: 'profile_compose_main_cpu_limit', minimumMemory: 256, minimumCPU: 0.35 }
	];

	let settings: NumericSettings = { ...defaultSettings };
	let savedSettings: NumericSettings = { ...defaultSettings };
	let hostStats: HostStats | null = null;
	let loadingSettings = true;
	let savingSettings = false;
	let triggeringUpdate = false;
	let updateOverlayOpen = false;
	let currentBuildSha = '';
	let updatePoll: ReturnType<typeof setInterval> | undefined;
	let updatePollTimeout: ReturnType<typeof setTimeout> | undefined;

	$: settingsChanged = (Object.keys(defaultSettings) as SettingKey[]).some((key) => settings[key] !== savedSettings[key]);
	$: validationErrors = {
		build_timeout_minutes: numberError(settings.build_timeout_minutes, 1, 1440, true, 'Use 1–1440 minutes.'),
		profile_static_memory_mb: numberError(settings.profile_static_memory_mb, 64, 32768, true, 'Minimum 64 MB.'),
		profile_static_cpu_limit: numberError(settings.profile_static_cpu_limit, 0.1, 32, false, 'Minimum 0.10 CPU.'),
		profile_go_small_memory_mb: numberError(settings.profile_go_small_memory_mb, 128, 32768, true, 'Minimum 128 MB.'),
		profile_go_small_cpu_limit: numberError(settings.profile_go_small_cpu_limit, 0.2, 32, false, 'Minimum 0.20 CPU.'),
		profile_node_python_memory_mb: numberError(settings.profile_node_python_memory_mb, 256, 32768, true, 'Minimum 256 MB.'),
		profile_node_python_cpu_limit: numberError(settings.profile_node_python_cpu_limit, 0.35, 32, false, 'Minimum 0.35 CPU.'),
		profile_compose_main_memory_mb: numberError(settings.profile_compose_main_memory_mb, 256, 32768, true, 'Minimum 256 MB.'),
		profile_compose_main_cpu_limit: numberError(settings.profile_compose_main_cpu_limit, 0.35, 32, false, 'Minimum 0.35 CPU.')
	};
	$: hasValidationErrors = Object.values(validationErrors).some(Boolean);
	$: hostMemoryTotal = hostStats?.memory?.total_bytes ?? hostStats?.host_ram_bytes ?? 0;
	$: hostMemoryUsed = hostStats?.memory ? Math.max(0, hostStats.memory.total_bytes - hostStats.memory.available_bytes) : 0;
	$: hostStorageUsed = hostStats?.storage ? Math.max(0, hostStats.storage.total_bytes - hostStats.storage.available_bytes) : 0;

	onMount(() => {
		void loadSettings();
	});
	onDestroy(() => {
		if (updatePoll) clearInterval(updatePoll);
		if (updatePollTimeout) clearTimeout(updatePollTimeout);
	});

	async function loadSettings() {
		loadingSettings = true;
		try {
			const [data, capacity] = await Promise.all([
				api.admin.getSettings(),
				api.admin.getHostStats().catch(() => null)
			]);
			settings = {
				build_timeout_minutes: numericValue(data.build_timeout_minutes, defaultSettings.build_timeout_minutes),
				profile_static_memory_mb: numericValue(data.profile_static_memory_mb, 64),
				profile_static_cpu_limit: numericValue(data.profile_static_cpu_limit, 0.1),
				profile_go_small_memory_mb: numericValue(data.profile_go_small_memory_mb, 128),
				profile_go_small_cpu_limit: numericValue(data.profile_go_small_cpu_limit, 0.2),
				profile_node_python_memory_mb: numericValue(data.profile_node_python_memory_mb, 256),
				profile_node_python_cpu_limit: numericValue(data.profile_node_python_cpu_limit, 0.35),
				profile_compose_main_memory_mb: numericValue(data.profile_compose_main_memory_mb, 256),
				profile_compose_main_cpu_limit: numericValue(data.profile_compose_main_cpu_limit, 0.35)
			};
			currentBuildSha = ((data as any).build_sha as string) || '';
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
			settings = Object.fromEntries(
				(Object.keys(defaultSettings) as SettingKey[]).map((key) => [key, numericValue(updated[key], settings[key])])
			) as NumericSettings;
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
		triggeringUpdate = true;
		try {
			await api.admin.triggerUpdate();
			updateOverlayOpen = true;
			startUpdatePolling();
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to update MyPaaS');
			console.error(error);
		} finally {
			triggeringUpdate = false;
		}
	}

	function startUpdatePolling() {
		let wasDown = false;
		if (updatePoll) clearInterval(updatePoll);
		if (updatePollTimeout) clearTimeout(updatePollTimeout);
		updatePoll = setInterval(async () => {
			try {
				const res = await fetch('/api/health');
				if (res.ok) {
					if (wasDown) {
						if (updatePoll) clearInterval(updatePoll);
						window.location.href = '/';
					}
				} else {
					wasDown = true;
				}
			} catch {
				wasDown = true;
			}
		}, 3000);
		updatePollTimeout = setTimeout(() => {
			if (updatePoll) clearInterval(updatePoll);
			updatePoll = undefined;
			updateOverlayOpen = false;
			toast.info('No restart detected');
			void loadSettings();
		}, 120_000);
	}

	function discardChanges() {
		settings = { ...savedSettings };
	}

	function numericValue(value: unknown, fallback = 0) {
		return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
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
	<title>Settings · MyPaaS</title>
</svelte:head>

{#if updateOverlayOpen}
	<div class="fixed inset-0 z-50 flex flex-col items-center justify-center bg-white/90 backdrop-blur-sm dark:bg-gray-950/90">
		<LoadingIndicator label="Updating MyPaaS" size="lg" />
		<p class="mt-4 text-sm text-gray-500 dark:text-gray-400">Waiting for MyPaaS to restart.</p>
	</div>
{/if}

<div class="page-shell">
	{#if loadingSettings}
		<div class="flex min-h-48 items-center justify-center"><LoadingIndicator label="Loading settings" /></div>
	{:else}
		<div class="max-w-5xl">
			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5"><h2 class="text-xs font-semibold uppercase tracking-[0.08em] text-gray-500 dark:text-gray-400">Host</h2></div>
				<div class="grid border-t border-[color:var(--workspace-divider)] sm:grid-cols-3">
					<div class="flex min-w-0 items-start gap-2.5 px-4 py-3 sm:border-r sm:border-[color:var(--workspace-divider)]">
						<MemoryStick class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
						<div class="min-w-0"><p class="text-xs text-gray-500 dark:text-gray-400">Memory</p><p class="mt-0.5 text-sm font-semibold text-gray-950 dark:text-white">{hostStats ? formatBytes(hostMemoryTotal) : 'Unavailable'}</p>{#if hostStats}<p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">{formatBytes(hostStats.allocated_ram_mb * 1024 * 1024)} allocated{hostStats.memory ? ` · ${formatBytes(hostMemoryUsed)} used` : ''}</p>{/if}</div>
					</div>
					<div class="flex min-w-0 items-start gap-2.5 border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:border-t-0 sm:border-r">
						<Cpu class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
						<div class="min-w-0"><p class="text-xs text-gray-500 dark:text-gray-400">CPU</p><p class="mt-0.5 text-sm font-semibold text-gray-950 dark:text-white">{hostStats ? `${hostStats.host_cpu_cores} core${hostStats.host_cpu_cores === 1 ? '' : 's'}` : 'Unavailable'}</p>{#if hostStats}<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{hostStats.allocated_cpu.toFixed(2)} allocated</p>{/if}</div>
					</div>
					<div class="flex min-w-0 items-start gap-2.5 border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:border-t-0">
						<HardDrive class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
						<div class="min-w-0"><p class="text-xs text-gray-500 dark:text-gray-400">Storage</p><p class="mt-0.5 text-sm font-semibold text-gray-950 dark:text-white">{hostStats?.storage ? formatBytes(hostStats.storage.total_bytes) : 'Unavailable'}</p>{#if hostStats?.storage}<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{formatBytes(hostStorageUsed)} used · {formatBytes(hostStats.storage.available_bytes)} free</p>{/if}</div>
					</div>
				</div>
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5"><h2 class="text-xs font-semibold uppercase tracking-[0.08em] text-gray-500 dark:text-gray-400">Resource defaults</h2></div>
				<div class="border-t border-[color:var(--workspace-divider)]">
					{#each profileSettings as profile, index}
						<div class={`grid gap-3 px-4 py-2.5 sm:grid-cols-[1.25rem_9rem_11rem_11rem] sm:items-start ${index > 0 ? 'border-t border-[color:var(--workspace-divider)]' : ''}`}>
							<SlidersHorizontal class="mt-7 h-4 w-4 text-gray-400 dark:text-gray-500 sm:mt-2.5" aria-hidden="true" />
							<p class="pt-2 text-sm font-medium text-gray-950 dark:text-white">{profile.name}</p>
							<label class="block" for={profile.memoryKey}>
								<span class="mb-1 block text-xs text-gray-500 dark:text-gray-400">Memory</span>
								<div class="relative"><input type="number" id={profile.memoryKey} min={profile.minimumMemory} max="32768" step="1" bind:value={settings[profile.memoryKey]} class="field compact-number-input w-full pr-11" aria-invalid={validationErrors[profile.memoryKey] ? 'true' : undefined} /><span class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-gray-500 dark:text-gray-400">MB</span></div>
								{#if validationErrors[profile.memoryKey]}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors[profile.memoryKey]}</p>{/if}
							</label>
							<label class="block" for={profile.cpuKey}>
								<span class="mb-1 block text-xs text-gray-500 dark:text-gray-400">CPU</span>
								<div class="relative"><input type="number" id={profile.cpuKey} min={profile.minimumCPU} max="32" step="0.05" bind:value={settings[profile.cpuKey]} class="field compact-number-input w-full pr-12" aria-invalid={validationErrors[profile.cpuKey] ? 'true' : undefined} /><span class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-gray-500 dark:text-gray-400">CPU</span></div>
								{#if validationErrors[profile.cpuKey]}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors[profile.cpuKey]}</p>{/if}
							</label>
						</div>
					{/each}
				</div>
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5"><h2 class="text-xs font-semibold uppercase tracking-[0.08em] text-gray-500 dark:text-gray-400">Deployment</h2></div>
				<div class="grid gap-3 border-t border-[color:var(--workspace-divider)] px-4 py-2.5 sm:grid-cols-[1.25rem_9rem_11rem] sm:items-start">
					<Timer class="mt-2.5 h-4 w-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					<label class="pt-2 text-sm text-gray-600 dark:text-gray-400" for="build_timeout_minutes">Build timeout</label>
					<div><div class="relative"><input type="number" id="build_timeout_minutes" min="1" max="1440" step="1" bind:value={settings.build_timeout_minutes} class="field compact-number-input w-full pr-16" aria-invalid={validationErrors.build_timeout_minutes ? 'true' : undefined} /><span class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-gray-500 dark:text-gray-400">minutes</span></div>{#if validationErrors.build_timeout_minutes}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.build_timeout_minutes}</p>{/if}</div>
				</div>
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5"><h2 class="text-xs font-semibold uppercase tracking-[0.08em] text-gray-500 dark:text-gray-400">System update</h2></div>
				<div class="grid gap-3 border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:grid-cols-[1.25rem_9rem_minmax(0,1fr)_auto] sm:items-center">
					<RefreshCw class="h-4 w-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					<p class="text-sm text-gray-600 dark:text-gray-400">Current build</p>
					<div class="min-w-0"><p class="font-mono text-sm font-semibold text-gray-950 dark:text-white">{currentBuildSha ? currentBuildSha.substring(0, 12) : 'Unknown'}</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Installs a newer release when available. MyPaaS may restart.</p></div>
					<ActionButton variant="secondary" size="sm" loading={triggeringUpdate} loadingLabel="Updating" on:click={triggerUpdate}>Update MyPaaS</ActionButton>
				</div>
			</section>

			{#if settingsChanged}
				<div class="flex flex-wrap items-center justify-end gap-2 px-4 py-3">
					<ActionButton variant="ghost" size="sm" on:click={discardChanges} disabled={savingSettings}>Discard</ActionButton>
					<ActionButton variant="primary" size="sm" loading={savingSettings} loadingLabel="Saving" on:click={saveSettings} disabled={hasValidationErrors}>Save changes</ActionButton>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	:global(.compact-number-input) {
		-moz-appearance: textfield;
		appearance: textfield;
	}

	:global(.compact-number-input::-webkit-inner-spin-button),
	:global(.compact-number-input::-webkit-outer-spin-button) {
		margin: 0;
		-webkit-appearance: none;
	}
</style>
