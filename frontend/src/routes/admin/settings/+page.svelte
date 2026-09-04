<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { Cpu, HardDrive, MemoryStick, Pencil, RefreshCw, Timer } from '@lucide/svelte';
	import { api, type HostStats } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import ConfirmActionDialog from '$components/ConfirmActionDialog.svelte';
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
		id: string;
		name: string;
		description: string;
		memoryKey: Extract<SettingKey, `${string}_memory_mb`>;
		cpuKey: Extract<SettingKey, `${string}_cpu_limit`>;
		minimumMemory: number;
		minimumCPU: number;
	};
	type ConfirmationTarget =
		| { kind: 'profile'; profile: ProfileSetting }
		| { kind: 'build-timeout' }
		| { kind: 'system-update' }
		| null;

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
		{ id: 'static', name: 'Static', description: 'Static and no-runtime projects', memoryKey: 'profile_static_memory_mb', cpuKey: 'profile_static_cpu_limit', minimumMemory: 64, minimumCPU: 0.1 },
		{ id: 'go-small', name: 'Go small', description: 'Small Go services', memoryKey: 'profile_go_small_memory_mb', cpuKey: 'profile_go_small_cpu_limit', minimumMemory: 128, minimumCPU: 0.2 },
		{ id: 'node-python', name: 'Node / Python', description: 'Node.js and Python applications', memoryKey: 'profile_node_python_memory_mb', cpuKey: 'profile_node_python_cpu_limit', minimumMemory: 256, minimumCPU: 0.35 },
		{ id: 'compose-main', name: 'Compose main', description: 'Primary Docker Compose service', memoryKey: 'profile_compose_main_memory_mb', cpuKey: 'profile_compose_main_cpu_limit', minimumMemory: 256, minimumCPU: 0.35 }
	];

	let settings: NumericSettings = { ...defaultSettings };
	let savedSettings: NumericSettings = { ...defaultSettings };
	let hostStats: HostStats | null = null;
	let loadingSettings = true;
	let savingTarget = '';
	let editingTarget = '';
	let confirmationTarget: ConfirmationTarget = null;
	let triggeringUpdate = false;
	let updateOverlayOpen = false;
	let currentBuildSha = '';
	let updatePoll: ReturnType<typeof setInterval> | undefined;
	let updatePollTimeout: ReturnType<typeof setTimeout> | undefined;

	$: validationErrors = {
		build_timeout_minutes: numberError(settings.build_timeout_minutes, 1, 1440, true, 'Use a whole number from 1 to 1440 minutes.'),
		profile_static_memory_mb: numberError(settings.profile_static_memory_mb, 64, 32768, true, 'Use 64–32768 MB.'),
		profile_static_cpu_limit: numberError(settings.profile_static_cpu_limit, 0.1, 32, false, 'Use 0.10–32 CPU.'),
		profile_go_small_memory_mb: numberError(settings.profile_go_small_memory_mb, 128, 32768, true, 'Use 128–32768 MB.'),
		profile_go_small_cpu_limit: numberError(settings.profile_go_small_cpu_limit, 0.2, 32, false, 'Use 0.20–32 CPU.'),
		profile_node_python_memory_mb: numberError(settings.profile_node_python_memory_mb, 256, 32768, true, 'Use 256–32768 MB.'),
		profile_node_python_cpu_limit: numberError(settings.profile_node_python_cpu_limit, 0.35, 32, false, 'Use 0.35–32 CPU.'),
		profile_compose_main_memory_mb: numberError(settings.profile_compose_main_memory_mb, 256, 32768, true, 'Use 256–32768 MB.'),
		profile_compose_main_cpu_limit: numberError(settings.profile_compose_main_cpu_limit, 0.35, 32, false, 'Use 0.35–32 CPU.')
	};
	$: hostMemoryTotal = hostStats?.memory?.total_bytes ?? hostStats?.host_ram_bytes ?? 0;
	$: hostMemoryUsed = hostStats?.memory ? Math.max(0, hostStats.memory.total_bytes - hostStats.memory.available_bytes) : 0;
	$: hostStorageUsed = hostStats?.storage ? Math.max(0, hostStats.storage.total_bytes - hostStats.storage.available_bytes) : 0;
	$: hostCPUAllocatedPercent = hostStats ? percentage(hostStats.allocated_cpu, hostStats.host_cpu_cores) : 0;
	$: hostMemoryUsedPercent = percentage(hostMemoryUsed, hostMemoryTotal);
	$: hostStorageUsedPercent = hostStats?.storage ? percentage(hostStorageUsed, hostStats.storage.total_bytes) : 0;
	$: confirmationTitle = confirmationTarget?.kind === 'profile'
		? `Save ${confirmationTarget.profile.name} defaults?`
		: confirmationTarget?.kind === 'build-timeout'
			? 'Save build timeout?'
			: confirmationTarget?.kind === 'system-update'
				? 'Update MyPaaS?'
				: 'Confirm action';
	$: confirmationDescription = confirmationTarget?.kind === 'profile'
		? 'This becomes the platform default used by this resource profile. Explicit per-project overrides remain available.'
		: confirmationTarget?.kind === 'build-timeout'
			? 'Future deployment builds will use this timeout.'
			: confirmationTarget?.kind === 'system-update'
				? 'This queues the host updater. MyPaaS may restart while a newer published revision is applied.'
				: '';
	$: confirmationLabel = confirmationTarget?.kind === 'system-update' ? 'Update MyPaaS' : 'Save changes';
	$: confirmationBusy = confirmationTarget?.kind === 'system-update' ? triggeringUpdate : Boolean(savingTarget);

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

	function beginProfileEdit(profile: ProfileSetting) {
		cancelEdit();
		editingTarget = `profile:${profile.id}`;
	}

	function beginBuildTimeoutEdit() {
		cancelEdit();
		editingTarget = 'build-timeout';
	}

	function cancelEdit() {
		settings = { ...savedSettings };
		editingTarget = '';
		confirmationTarget = null;
	}

	function profileChanged(profile: ProfileSetting) {
		return settings[profile.memoryKey] !== savedSettings[profile.memoryKey]
			|| settings[profile.cpuKey] !== savedSettings[profile.cpuKey];
	}

	function profileInvalid(profile: ProfileSetting) {
		return Boolean(validationErrors[profile.memoryKey] || validationErrors[profile.cpuKey]);
	}

	function requestProfileSave(profile: ProfileSetting) {
		if (!profileChanged(profile) || profileInvalid(profile) || savingTarget) return;
		confirmationTarget = { kind: 'profile', profile };
	}

	function requestBuildTimeoutSave() {
		if (settings.build_timeout_minutes === savedSettings.build_timeout_minutes || validationErrors.build_timeout_minutes || savingTarget) return;
		confirmationTarget = { kind: 'build-timeout' };
	}

	async function saveProfile(profile: ProfileSetting) {
		const target = `profile:${profile.id}`;
		if (savingTarget || profileInvalid(profile) || !profileChanged(profile)) return;
		savingTarget = target;
		try {
			const updated = await api.admin.updateSettings({
				[profile.memoryKey]: settings[profile.memoryKey],
				[profile.cpuKey]: settings[profile.cpuKey]
			});
			const memory = numericValue(updated[profile.memoryKey], settings[profile.memoryKey]);
			const cpu = numericValue(updated[profile.cpuKey], settings[profile.cpuKey]);
			settings = { ...settings, [profile.memoryKey]: memory, [profile.cpuKey]: cpu };
			savedSettings = { ...savedSettings, [profile.memoryKey]: memory, [profile.cpuKey]: cpu };
			editingTarget = '';
			confirmationTarget = null;
			toast.success(`${profile.name} defaults saved`);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to save resource defaults');
			console.error(error);
		} finally {
			savingTarget = '';
		}
	}

	async function saveBuildTimeout() {
		if (savingTarget || validationErrors.build_timeout_minutes || settings.build_timeout_minutes === savedSettings.build_timeout_minutes) return;
		savingTarget = 'build-timeout';
		try {
			const updated = await api.admin.updateSettings({ build_timeout_minutes: settings.build_timeout_minutes });
			const timeout = numericValue(updated.build_timeout_minutes, settings.build_timeout_minutes);
			settings = { ...settings, build_timeout_minutes: timeout };
			savedSettings = { ...savedSettings, build_timeout_minutes: timeout };
			editingTarget = '';
			confirmationTarget = null;
			toast.success('Build timeout saved');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to save build timeout');
			console.error(error);
		} finally {
			savingTarget = '';
		}
	}

	async function triggerUpdate() {
		if (triggeringUpdate) return;
		triggeringUpdate = true;
		try {
			await api.admin.triggerUpdate();
			confirmationTarget = null;
			updateOverlayOpen = true;
			startUpdatePolling();
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to update MyPaaS');
			console.error(error);
		} finally {
			triggeringUpdate = false;
		}
	}

	async function confirmPendingAction() {
		if (confirmationTarget?.kind === 'profile') {
			await saveProfile(confirmationTarget.profile);
			return;
		}
		if (confirmationTarget?.kind === 'build-timeout') {
			await saveBuildTimeout();
			return;
		}
		if (confirmationTarget?.kind === 'system-update') await triggerUpdate();
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

	function numericValue(value: unknown, fallback = 0) {
		return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
	}

	function numberError(value: number, min: number, max: number, integer: boolean, message: string) {
		if (!Number.isFinite(value) || value < min || value > max || (integer && !Number.isInteger(value))) return message;
		return '';
	}

	function percentage(value: number, total: number) {
		if (!Number.isFinite(value) || !Number.isFinite(total) || total <= 0) return 0;
		return Math.min(100, Math.max(0, (value / total) * 100));
	}

	function formatCPU(value: number) {
		return Number.isInteger(value) ? String(value) : value.toFixed(2).replace(/0+$/, '').replace(/\.$/, '');
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
		<p class="mt-4 text-sm text-gray-500 dark:text-gray-400">Update request accepted. Waiting for MyPaaS to restart.</p>
	</div>
{/if}

<div class="page-shell">
	{#if loadingSettings}
		<div class="flex min-h-48 items-center justify-center"><LoadingIndicator label="Loading settings" /></div>
	{:else}
		<div class="admin-general-workspace w-full border-y border-[color:var(--workspace-divider)]">
			<section>
				<div class="flex items-start justify-between gap-4 px-4 py-3">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Host capacity</h2>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Current host capacity and project allocations.</p>
					</div>
				</div>
				<div class="grid border-t border-[color:var(--workspace-divider)] sm:grid-cols-3">
					<div class="min-w-0 px-4 py-3 sm:border-r sm:border-[color:var(--workspace-divider)]">
						<div class="flex items-start gap-2.5">
							<MemoryStick class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0 flex-1">
								<p class="text-xs text-gray-500 dark:text-gray-400">Memory</p>
								<p class="mt-0.5 text-base font-semibold tabular-nums text-gray-950 dark:text-white">{hostStats ? formatBytes(hostMemoryTotal) : 'Unavailable'}</p>
								{#if hostStats}<p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">{formatBytes(hostStats.allocated_ram_mb * 1024 * 1024)} allocated{hostStats.memory ? ` · ${formatBytes(hostMemoryUsed)} used` : ''}</p>{/if}
							</div>
						</div>
						<div class="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-neutral-800"><div class="h-full rounded-full" style={`width:${hostMemoryUsedPercent}%; background:var(--chart-memory);`}></div></div>
					</div>
					<div class="min-w-0 border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:border-t-0 sm:border-r">
						<div class="flex items-start gap-2.5">
							<Cpu class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0 flex-1">
								<p class="text-xs text-gray-500 dark:text-gray-400">CPU</p>
								<p class="mt-0.5 text-base font-semibold tabular-nums text-gray-950 dark:text-white">{hostStats ? `${hostStats.host_cpu_cores} core${hostStats.host_cpu_cores === 1 ? '' : 's'}` : 'Unavailable'}</p>
								{#if hostStats}<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{hostStats.allocated_cpu.toFixed(2)} allocated</p>{/if}
							</div>
						</div>
						<div class="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-neutral-800"><div class="h-full rounded-full" style={`width:${hostCPUAllocatedPercent}%; background:var(--chart-cpu);`}></div></div>
					</div>
					<div class="min-w-0 border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:border-t-0">
						<div class="flex items-start gap-2.5">
							<HardDrive class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0 flex-1">
								<p class="text-xs text-gray-500 dark:text-gray-400">Storage</p>
								<p class="mt-0.5 text-base font-semibold tabular-nums text-gray-950 dark:text-white">{hostStats?.storage ? formatBytes(hostStats.storage.total_bytes) : 'Unavailable'}</p>
								{#if hostStats?.storage}<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{formatBytes(hostStorageUsed)} used · {formatBytes(hostStats.storage.available_bytes)} free</p>{/if}
							</div>
						</div>
						<div class="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-neutral-800"><div class="h-full rounded-full" style={`width:${hostStorageUsedPercent}%; background:var(--chart-storage);`}></div></div>
					</div>
				</div>
			</section>

			<section class="border-t border-[color:var(--workspace-divider)]">
				<div class="flex items-start justify-between gap-4 px-4 py-3">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Resource defaults</h2>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Defaults used by new projects. Built-in profile floors cannot be lowered.</p>
					</div>
				</div>
				<div class="grid border-t border-[color:var(--workspace-divider)] lg:grid-cols-2">
					{#each profileSettings as profile, index}
						<div class={`min-w-0 px-4 py-3 ${index >= 2 ? 'border-t border-[color:var(--workspace-divider)]' : ''} ${index % 2 === 1 ? 'lg:border-l lg:border-[color:var(--workspace-divider)]' : ''} ${index === 1 ? 'border-t border-[color:var(--workspace-divider)] lg:border-t-0' : ''}`}>
							<div class="flex min-h-9 items-start justify-between gap-3">
								<div class="min-w-0">
									<p class="text-sm font-semibold text-gray-950 dark:text-white">{profile.name}</p>
									<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{profile.description}</p>
								</div>
								{#if editingTarget !== `profile:${profile.id}`}
									<ActionButton variant="ghost" size="xs" on:click={() => beginProfileEdit(profile)} disabled={Boolean(savingTarget)}><Pencil slot="icon" class="h-3.5 w-3.5" />Edit</ActionButton>
								{/if}
							</div>

							{#if editingTarget === `profile:${profile.id}`}
								<div class="mt-3 grid gap-3 sm:grid-cols-2">
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
								<div class="mt-3 flex justify-end gap-2">
									<ActionButton variant="ghost" size="xs" on:click={cancelEdit} disabled={Boolean(savingTarget)}>Cancel</ActionButton>
									<ActionButton variant="primary" size="xs" on:click={() => requestProfileSave(profile)} disabled={!profileChanged(profile) || profileInvalid(profile) || Boolean(savingTarget)}>Save</ActionButton>
								</div>
							{:else}
								<div class="mt-3 grid grid-cols-2 divide-x divide-[color:var(--workspace-divider)] border-t border-[color:var(--workspace-divider)] pt-2.5">
									<div class="pr-4"><p class="text-xs text-gray-500 dark:text-gray-400">Memory</p><p class="mt-0.5 text-sm font-semibold tabular-nums text-gray-950 dark:text-white">{savedSettings[profile.memoryKey]} MB</p></div>
									<div class="pl-4"><p class="text-xs text-gray-500 dark:text-gray-400">CPU</p><p class="mt-0.5 text-sm font-semibold tabular-nums text-gray-950 dark:text-white">{formatCPU(savedSettings[profile.cpuKey])} CPU</p></div>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</section>

			<div class="grid border-t border-[color:var(--workspace-divider)] xl:grid-cols-2">
				<section class="min-w-0 xl:border-r xl:border-[color:var(--workspace-divider)]">
					<div class="flex items-start justify-between gap-4 px-4 py-3">
						<div>
							<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Build defaults</h2>
							<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Maximum duration allowed for an application build.</p>
						</div>
					</div>
					<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3">
						<div class="flex items-center gap-3">
							<Timer class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0 flex-1">
								<p class="text-xs text-gray-500 dark:text-gray-400">Build timeout</p>
								{#if editingTarget === 'build-timeout'}
									<div class="mt-1.5 max-w-xs">
										<div class="relative"><input type="number" id="build_timeout_minutes" min="1" max="1440" step="1" bind:value={settings.build_timeout_minutes} class="field compact-number-input w-full pr-16" aria-invalid={validationErrors.build_timeout_minutes ? 'true' : undefined} /><span class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-gray-500 dark:text-gray-400">minutes</span></div>
										{#if validationErrors.build_timeout_minutes}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.build_timeout_minutes}</p>{/if}
									</div>
								{:else}
									<p class="mt-0.5 text-base font-semibold tabular-nums text-gray-950 dark:text-white">{savedSettings.build_timeout_minutes} minutes</p>
								{/if}
							</div>
							{#if editingTarget === 'build-timeout'}
								<div class="flex gap-2"><ActionButton variant="ghost" size="xs" on:click={cancelEdit} disabled={Boolean(savingTarget)}>Cancel</ActionButton><ActionButton variant="primary" size="xs" on:click={requestBuildTimeoutSave} disabled={settings.build_timeout_minutes === savedSettings.build_timeout_minutes || Boolean(validationErrors.build_timeout_minutes) || Boolean(savingTarget)}>Save</ActionButton></div>
							{:else}
								<ActionButton variant="ghost" size="xs" on:click={beginBuildTimeoutEdit} disabled={Boolean(savingTarget)}><Pencil slot="icon" class="h-3.5 w-3.5" />Edit</ActionButton>
							{/if}
						</div>
					</div>
				</section>

				<section class="min-w-0 border-t border-[color:var(--workspace-divider)] xl:border-t-0">
					<div class="flex items-start justify-between gap-4 px-4 py-3">
						<div>
							<h2 class="text-sm font-semibold text-gray-950 dark:text-white">System update</h2>
							<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Ask the host updater to apply a newer published revision when available.</p>
						</div>
					</div>
					<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3">
						<div class="flex items-center gap-3">
							<RefreshCw class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0 flex-1">
								<p class="text-xs text-gray-500 dark:text-gray-400">Current build</p>
								<p class="mt-0.5 font-mono text-sm font-semibold text-gray-950 dark:text-white">{currentBuildSha ? currentBuildSha.substring(0, 12) : 'Unknown'}</p>
							</div>
							<ActionButton variant="secondary" size="sm" on:click={() => (confirmationTarget = { kind: 'system-update' })} disabled={Boolean(savingTarget)}>Update MyPaaS</ActionButton>
						</div>
					</div>
				</section>
			</div>
		</div>
	{/if}
</div>

<ConfirmActionDialog
	open={confirmationTarget !== null}
	title={confirmationTitle}
	description={confirmationDescription}
	confirmLabel={confirmationLabel}
	busy={confirmationBusy}
	busyLabel={confirmationTarget?.kind === 'system-update' ? 'Queuing update' : 'Saving'}
	on:cancel={() => !confirmationBusy && (confirmationTarget = null)}
	on:confirm={confirmPendingAction}
>
	{#if confirmationTarget?.kind === 'profile'}
		<div class="grid grid-cols-2 gap-4">
			<div><p class="text-xs text-gray-500 dark:text-gray-400">Memory</p><p class="mt-0.5 font-semibold tabular-nums text-gray-950 dark:text-white">{savedSettings[confirmationTarget.profile.memoryKey]} → {settings[confirmationTarget.profile.memoryKey]} MB</p></div>
			<div><p class="text-xs text-gray-500 dark:text-gray-400">CPU</p><p class="mt-0.5 font-semibold tabular-nums text-gray-950 dark:text-white">{formatCPU(savedSettings[confirmationTarget.profile.cpuKey])} → {formatCPU(settings[confirmationTarget.profile.cpuKey])} CPU</p></div>
		</div>
	{:else if confirmationTarget?.kind === 'build-timeout'}
		<p><span class="text-gray-500 dark:text-gray-400">Build timeout:</span> <span class="font-semibold tabular-nums text-gray-950 dark:text-white">{savedSettings.build_timeout_minutes} → {settings.build_timeout_minutes} minutes</span></p>
	{:else if confirmationTarget?.kind === 'system-update'}
		<p>The request is handed to the host-level MyPaaS updater; the dashboard or API may briefly restart.</p>
	{/if}
</ConfirmActionDialog>

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
