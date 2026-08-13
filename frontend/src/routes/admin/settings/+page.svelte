<script lang="ts">
	import { onMount } from 'svelte';
	import { LoaderCircle, Save } from '@lucide/svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';

	let settings: Record<string, number> = {
		user_ram_quota_gb: 0,
		user_cpu_quota: 0,
		max_projects: 0,
		max_concurrent_deploys: 0,
		project_default_ram_mb: 0,
		project_default_cpu: 0,
		build_timeout_minutes: 0
	};
	let loadingSettings = true;
	let savingSettings = false;

	const settingsConfig = [
		{ key: 'user_ram_quota_gb', label: 'User RAM quota', unit: 'GB' },
		{ key: 'user_cpu_quota', label: 'User CPU quota', unit: 'cores' },
		{ key: 'max_projects', label: 'Maximum projects', unit: 'projects' },
		{ key: 'max_concurrent_deploys', label: 'Concurrent deploys', unit: 'deploys' },
		{ key: 'project_default_ram_mb', label: 'Default project RAM', unit: 'MB' },
		{ key: 'project_default_cpu', label: 'Default project CPU', unit: 'cores' },
		{ key: 'build_timeout_minutes', label: 'Build timeout', unit: 'minutes' }
	];

	onMount(loadSettings);

	async function loadSettings() {
		loadingSettings = true;
		try {
			const data = await api.admin.getSettings();
			const numericSettings = Object.fromEntries(
				Object.entries(data).filter(([, value]) => typeof value === 'number')
			) as Record<string, number>;
			settings = { ...settings, ...numericSettings };
		} catch (error) {
			toast.error('Failed to load settings');
			console.error(error);
		} finally {
			loadingSettings = false;
		}
	}

	async function saveSettings() {
		if (savingSettings) return;
		savingSettings = true;
		try {
			const updated = await api.admin.updateSettings(settings);
			settings = { ...settings, ...updated };
			toast.success('Settings saved successfully');
		} catch (error) {
			toast.error('Failed to save settings');
			console.error(error);
		} finally {
			savingSettings = false;
		}
	}
</script>

<svelte:head>
	<title>Settings · MyPaas</title>
</svelte:head>

<div class="page-shell space-y-4 py-6">
	<p class="px-5 text-sm text-gray-500 dark:text-gray-400">Configure platform capacity limits and the defaults applied when new projects are created.</p>

	<SectionPanel title="Resource configuration" description="Default platform limits and project resource quotas.">
		<svelte:fragment slot="actions">
			<ActionButton variant="primary" size="sm" loading={savingSettings} loadingLabel="Saving" on:click={saveSettings} disabled={loadingSettings}>
				<Save slot="icon" class="h-4 w-4" />
				Save changes
			</ActionButton>
		</svelte:fragment>

		{#if loadingSettings}
			<div class="flex h-28 items-center justify-center">
				<LoaderCircle class="h-6 w-6 animate-spin motion-reduce:animate-none text-gray-500 dark:text-gray-400" aria-hidden="true" />
			</div>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
				{#each settingsConfig as { key, label, unit }}
					<label class="block" for={key}>
						<span class="field-label">{label}</span>
						<div class="flex items-center gap-2">
							<input type="number" id={key} bind:value={settings[key]} class="field min-w-0 flex-1" />
							<span class="w-16 shrink-0 text-xs text-gray-500 dark:text-gray-400">{unit}</span>
						</div>
					</label>
				{/each}
			</div>
		{/if}
	</SectionPanel>
</div>
