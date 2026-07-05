<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import { projectHost } from '$lib/utils/urls';
	import type { DeployModeDetection, EnvVarDiscovery, ResourceProfile } from '$types';

	type DeployModeChoice = 'auto' | 'dockerfile' | 'compose' | 'static';
	type EnvDraft = EnvVarDiscovery & {
		value: string;
	};

	let step = 1;
	let submitting = false;
	let detecting = false;
	let error = '';
	let detectMessage = '';
	let detectedServices: string[] = [];
	let envDrafts: EnvDraft[] = [];
	let newEnvKey = '';
	let form = {
		name:        '',
		repoUrl:     '',
		branch:      'main',
		deployMode:  'auto' as DeployModeChoice,
		mainService: '',
		appPort:     '3000',
		resourceProfile: 'node-python' as ResourceProfile,
		memoryMb:    '256',
		cpuLimit:    '0.35',
		sharedPostgres: false
	};

	const steps = ['Repository', 'Runtime', 'Environment', 'Resources'];
	const deployModes: Array<{ id: DeployModeChoice; title: string; body: string }> = [
		{ id: 'auto',       title: 'Auto',       body: 'Detect runtime files' },
		{ id: 'dockerfile', title: 'Dockerfile', body: 'Single container app' },
		{ id: 'compose',    title: 'Compose',    body: 'Multi-service project' },
		{ id: 'static',     title: 'Static',     body: 'No runtime container' }
	];
	const resourceProfiles: Array<{ id: ResourceProfile; title: string; memoryMb: string; cpuLimit: string }> = [
		{ id: 'node-python',  title: 'Node/Python',       memoryMb: '256', cpuLimit: '0.35' },
		{ id: 'go-small',     title: 'Go small',          memoryMb: '128', cpuLimit: '0.2' },
		{ id: 'compose-main', title: 'Compose main',      memoryMb: '256', cpuLimit: '0.35' },
		{ id: 'static',       title: 'Static/no-runtime', memoryMb: '64',  cpuLimit: '0.1' },
		{ id: 'custom',       title: 'Custom',            memoryMb: '512', cpuLimit: '0.5' }
	];

	function next() { step = Math.min(step + 1, steps.length); }
	function back() { step = Math.max(step - 1, 1); }
	$: previewHost = projectHost(form.name || 'your-app', $page.url.hostname);
	$: selectedProfile = resourceProfiles.find((profile) => profile.id === form.resourceProfile);

	function defaultProfileForMode(mode: DeployModeChoice): ResourceProfile {
		if (mode === 'static') return 'static';
		return mode === 'compose' ? 'compose-main' : 'node-python';
	}

	function applyResourceProfile(id: ResourceProfile) {
		const profile = resourceProfiles.find((item) => item.id === id);
		if (!profile) return;
		form.resourceProfile = profile.id;
		form.memoryMb = profile.memoryMb;
		form.cpuLimit = profile.cpuLimit;
	}

	function chooseDeployMode(mode: DeployModeChoice) {
		form.deployMode = mode;
		if (mode === 'static') {
			form.appPort = '80';
			form.mainService = '';
			form.sharedPostgres = false;
		} else if (form.appPort === '80') {
			form.appPort = '3000';
		}
		if (form.resourceProfile !== 'custom') {
			applyResourceProfile(defaultProfileForMode(mode));
		}
	}

	function markCustomProfile() {
		form.resourceProfile = 'custom';
	}

	function mergeDiscoveredEnvVars(vars: EnvVarDiscovery[]) {
		const existing = new Set(envDrafts.map((item) => item.key));
		const nextDrafts = [...envDrafts];
		for (const item of vars) {
			if (!item.key || existing.has(item.key)) continue;
			nextDrafts.push({ ...item, value: '' });
			existing.add(item.key);
		}
		envDrafts = nextDrafts.sort((a, b) => a.key.localeCompare(b.key));
	}

	function addEnvVar() {
		const key = newEnvKey.trim();
		if (!key || envDrafts.some((item) => item.key === key)) {
			newEnvKey = '';
			return;
		}
		envDrafts = [...envDrafts, { key, source: 'manual', sensitive: isSensitiveEnvKey(key), value: '' }]
			.sort((a, b) => a.key.localeCompare(b.key));
		newEnvKey = '';
	}

	function removeEnvVar(index: number) {
		envDrafts = envDrafts.filter((_, itemIndex) => itemIndex !== index);
	}

	function isSensitiveEnvKey(key: string) {
		return /SECRET|TOKEN|PASSWORD|PASS|KEY|DATABASE_URL|DSN|PRIVATE/i.test(key);
	}

	$: managedDatabaseUrl = form.sharedPostgres && form.deployMode !== 'static';

	async function handleDetectMode(showToast = true): Promise<DeployModeDetection> {
		if (!form.repoUrl.trim()) {
			throw new Error('Repository URL is required before detection');
		}

		detecting = true;
		error = '';
		detectMessage = '';
		try {
			const detected = await api.projects.detectMode({
				repoUrl: form.repoUrl,
				branch: form.branch
			});
			form.deployMode = detected.deployMode;
			if (form.resourceProfile !== 'custom') {
				applyResourceProfile(defaultProfileForMode(detected.deployMode));
			}
			if (detected.mainService) {
				form.mainService = detected.mainService;
			}
			detectedServices = detected.services;
			mergeDiscoveredEnvVars(detected.envVars ?? []);
			detectMessage = detected.deployMode === 'compose'
				? `Detected Compose${detected.composeFile ? ` (${detected.composeFile})` : ''}`
				: detected.deployMode === 'static'
					? 'Detected static site'
					: 'Detected Dockerfile';
			if (showToast) {
				toast.success(detectMessage);
			}
			return detected;
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to detect deploy mode';
			error = message;
			if (showToast) {
				toast.error(message);
			}
			throw err;
		} finally {
			detecting = false;
		}
	}

	async function handleSubmit() {
		if (submitting) return;
		submitting = true;
		error = '';
		try {
			let deployMode = form.deployMode;
			let mainService = form.mainService || null;
			if (deployMode === 'auto') {
				const detected = await handleDetectMode(false);
				deployMode = detected.deployMode;
				mainService = detected.mainService || mainService;
			}
			if (deployMode === 'static') {
				mainService = null;
				form.appPort = '80';
				form.sharedPostgres = false;
			}
			const envVars = envDrafts
				.filter((item) => item.key.trim() && item.value.length > 0)
				.map((item) => ({
					key: item.key.trim(),
					value: item.value
				}));

			const project = await api.projects.create({
				name: form.name,
				repoUrl: form.repoUrl,
				branch: form.branch,
				deployMode,
				resourceProfile: form.resourceProfile,
				mainService,
				appPort: Number(form.appPort),
				memoryLimitMb: Number(form.memoryMb),
				cpuLimit: Number(form.cpuLimit),
				sharedPostgres: form.sharedPostgres,
				envVars
			});
			toast.success('Project created');
			await goto(`/projects/${project.id}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create project';
			toast.error(error);
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>New project · MyPaas</title>
</svelte:head>

<div class="mx-auto max-w-6xl px-4 py-7 sm:px-6">
	<a href="/projects" class="mb-5 inline-flex items-center gap-1.5 text-sm font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">
		<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
			<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
		</svg>
		Projects
	</a>

	<header class="mb-6">
		<p class="text-xs font-medium uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400">Create deployment target</p>
		<h1 class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">New project</h1>
		<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Connect a Git repository and choose the container runtime MyPaas should run.</p>
	</header>

	<div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_22rem]">
		<section class="surface overflow-hidden">
			<div class="grid grid-cols-4 border-b border-gray-100 bg-gray-50/70 dark:border-gray-800 dark:bg-gray-900/70">
				{#each steps as label, index}
					<button
						type="button"
						on:click={() => (step = index + 1)}
						class="border-r border-gray-100 px-4 py-3 text-left last:border-r-0 dark:border-gray-800
							{step === index + 1 ? 'bg-white dark:bg-gray-900' : 'text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white'}"
					>
						<span class="block text-[11px] font-medium uppercase tracking-wide">Step {index + 1}</span>
						<span class="mt-1 block text-sm font-semibold">{label}</span>
					</button>
				{/each}
			</div>

			<div class="p-5">
				{#if error}
					<div class="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-200">
						{error}
					</div>
				{/if}

				{#if step === 1}
					<div class="space-y-5">
						<div>
							<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Repository</h2>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">The repository root must contain a Dockerfile, Compose file, or static index.html output.</p>
						</div>
						<div class="grid gap-4 sm:grid-cols-2">
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="name">Project name</label>
								<input id="name" type="text" bind:value={form.name} placeholder="my-app" class="field w-full" />
							</div>
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="branch">Branch</label>
								<input id="branch" type="text" bind:value={form.branch} class="field w-full font-mono" />
							</div>
							<div class="sm:col-span-2">
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="repo">Repository URL</label>
								<input id="repo" type="text" bind:value={form.repoUrl} placeholder="https://github.com/username/repo" class="field w-full font-mono" />
							</div>
						</div>
					</div>
				{:else if step === 2}
					<div class="space-y-5">
						<div>
							<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Runtime</h2>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Auto-detect prefers Compose when both runtime files exist.</p>
						</div>
						<div class="grid gap-2 sm:grid-cols-4">
							{#each deployModes as mode}
								<button
									type="button"
									on:click={() => chooseDeployMode(mode.id)}
									class="rounded-md border p-3 text-left transition-colors
										{form.deployMode === mode.id
											? 'border-gray-950 bg-gray-950 text-white dark:border-white dark:bg-white dark:text-gray-950'
											: 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300 dark:hover:bg-gray-900'}"
								>
									<span class="block text-sm font-semibold">{mode.title}</span>
									<span class="mt-1 block text-xs opacity-70">{mode.body}</span>
								</button>
							{/each}
						</div>
						<div class="rounded-md border border-gray-200 bg-gray-50 p-3 dark:border-gray-800 dark:bg-gray-950">
							<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
								<p class="min-w-0 text-xs text-gray-500 dark:text-gray-400">
									{#if detectMessage}
										<span class="font-medium text-gray-950 dark:text-white">{detectMessage}</span>
										{#if detectedServices.length > 0}
											<span class="ml-1">Services: {detectedServices.join(', ')}</span>
										{/if}
									{:else}
										Run detection after entering the repository URL.
									{/if}
								</p>
								<ActionButton
									variant="secondary"
									size="xs"
									type="button"
									on:click={() => void handleDetectMode()}
									disabled={detecting || !form.repoUrl.trim()}
									loading={detecting}
									loadingLabel="Detecting..."
								>
									Detect
								</ActionButton>
							</div>
						</div>
						{#if form.deployMode === 'compose'}
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="mainService">Main service name</label>
								<input id="mainService" type="text" bind:value={form.mainService} placeholder="app" class="field w-full font-mono" />
							</div>
						{/if}
						{#if form.deployMode !== 'static'}
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="appPort">App port</label>
								<input id="appPort" type="number" bind:value={form.appPort} class="field w-full font-mono sm:w-44" />
							</div>
						{/if}
					</div>
				{:else if step === 3}
					<div class="space-y-5">
						<div>
							<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Environment</h2>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Detected keys become encrypted project env vars after create.</p>
						</div>
						{#if form.deployMode !== 'static'}
							<label class="flex items-start gap-3 rounded-md border border-gray-200 bg-white p-3 text-sm dark:border-gray-800 dark:bg-gray-950">
								<input type="checkbox" bind:checked={form.sharedPostgres} class="mt-1 h-4 w-4 rounded border-gray-300 text-gray-950 focus:ring-gray-950 dark:border-gray-700" />
								<span>
									<span class="block font-medium text-gray-950 dark:text-white">Provision shared PostgreSQL</span>
									<span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">MyPaas will create DATABASE_URL as a managed encrypted env var.</span>
								</span>
							</label>
						{/if}
						<div class="overflow-hidden rounded-md border border-gray-200 dark:border-gray-800">
							<div class="grid grid-cols-[minmax(8rem,1fr)_minmax(10rem,1.4fr)_6rem_2rem] gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2 text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-400">
								<span>Key</span>
								<span>Value</span>
								<span>Source</span>
								<span></span>
							</div>
							{#if managedDatabaseUrl}
								<div class="grid grid-cols-[minmax(8rem,1fr)_minmax(10rem,1.4fr)_6rem_2rem] items-center gap-2 border-b border-gray-100 px-3 py-2 dark:border-gray-800">
									<div class="min-w-0">
										<p class="truncate font-mono text-sm font-medium text-gray-950 dark:text-white">DATABASE_URL</p>
										<p class="text-[11px] text-gray-500 dark:text-gray-400">managed</p>
									</div>
									<input value="Generated on create" disabled class="field w-full opacity-70" />
									<span class="truncate text-xs text-gray-500 dark:text-gray-400">shared db</span>
									<span></span>
								</div>
							{/if}
							{#each envDrafts as draft, index}
								<div class="grid grid-cols-[minmax(8rem,1fr)_minmax(10rem,1.4fr)_6rem_2rem] items-center gap-2 border-b border-gray-100 px-3 py-2 last:border-b-0 dark:border-gray-800">
									<div class="min-w-0">
										<input bind:value={draft.key} class="field w-full font-mono" />
										{#if draft.sensitive}
											<p class="mt-1 text-[11px] text-gray-500 dark:text-gray-400">masked</p>
										{/if}
									</div>
									<input
										type={draft.sensitive ? 'password' : 'text'}
										bind:value={draft.value}
										placeholder={draft.defaultValue ? `sample: ${draft.defaultValue}` : ''}
										class="field w-full font-mono"
									/>
									<span class="truncate text-xs text-gray-500 dark:text-gray-400" title={draft.source}>{draft.source}</span>
									<button type="button" on:click={() => removeEnvVar(index)} class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-950 dark:hover:bg-gray-800 dark:hover:text-white">
										×
									</button>
								</div>
							{/each}
						</div>
						<div class="flex gap-2">
							<input bind:value={newEnvKey} placeholder="ENV_KEY" class="field min-w-0 flex-1 font-mono" on:keydown={(event) => event.key === 'Enter' && addEnvVar()} />
							<ActionButton type="button" variant="secondary" on:click={addEnvVar}>
								Add
							</ActionButton>
						</div>
					</div>
				{:else}
					<div class="space-y-5">
						<div>
							<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Resources</h2>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Keep defaults small for personal VM capacity.</p>
						</div>
						<div class="grid gap-4 sm:grid-cols-2">
							<div class="sm:col-span-2">
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="profile">Resource profile</label>
								<select
									id="profile"
									bind:value={form.resourceProfile}
									on:change={() => applyResourceProfile(form.resourceProfile)}
									class="field w-full"
								>
									{#each resourceProfiles as profile}
										<option value={profile.id}>{profile.title} ({profile.memoryMb} MB / {profile.cpuLimit} CPU)</option>
									{/each}
								</select>
							</div>
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="memory">Memory limit</label>
								<select id="memory" bind:value={form.memoryMb} on:change={markCustomProfile} class="field w-full">
									<option value="64">64 MB</option>
									<option value="128">128 MB</option>
									<option value="256">256 MB</option>
									<option value="512">512 MB</option>
									<option value="1024">1024 MB</option>
									<option value="2048">2048 MB</option>
								</select>
							</div>
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="cpu">CPU limit</label>
								<select id="cpu" bind:value={form.cpuLimit} on:change={markCustomProfile} class="field w-full">
									<option value="0.1">0.10 cores</option>
									<option value="0.2">0.20 cores</option>
									<option value="0.25">0.25 cores</option>
									<option value="0.35">0.35 cores</option>
									<option value="0.5">0.5 cores</option>
									<option value="1">1 core</option>
									<option value="2">2 cores</option>
								</select>
							</div>
						</div>
					</div>
				{/if}
			</div>

			<div class="flex items-center justify-between border-t border-gray-100 bg-gray-50/70 px-5 py-4 dark:border-gray-800 dark:bg-gray-900/70">
				<ActionButton type="button" on:click={back} disabled={step === 1}>
					Back
				</ActionButton>
				{#if step < steps.length}
					<ActionButton variant="primary" type="button" on:click={next}>
						Continue
					</ActionButton>
				{:else}
					<ActionButton
						variant="primary"
						type="button"
						on:click={handleSubmit}
						loading={submitting}
						loadingLabel="Creating..."
					>
						Create project
					</ActionButton>
				{/if}
			</div>
		</section>

		<aside class="surface h-fit overflow-hidden">
			<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800">
				<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Deployment plan</h2>
			</div>
			<dl class="divide-y divide-gray-100 text-sm dark:divide-gray-800">
				<div class="px-5 py-3">
					<dt class="text-xs text-gray-500 dark:text-gray-400">Subdomain</dt>
					<dd class="mt-1 truncate font-mono font-medium text-gray-950 dark:text-white">{previewHost}</dd>
				</div>
				<div class="px-5 py-3">
					<dt class="text-xs text-gray-500 dark:text-gray-400">Repository</dt>
					<dd class="mt-1 truncate font-mono text-gray-950 dark:text-white">{form.repoUrl || '-'}</dd>
				</div>
				<div class="grid grid-cols-2 divide-x divide-gray-100 dark:divide-gray-800">
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Branch</dt>
						<dd class="mt-1 font-mono text-gray-950 dark:text-white">{form.branch}</dd>
					</div>
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Runtime</dt>
						<dd class="mt-1 font-mono text-gray-950 dark:text-white">{form.deployMode}</dd>
					</div>
				</div>
				<div class="grid grid-cols-3 divide-x divide-gray-100 dark:divide-gray-800">
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Port</dt>
						<dd class="mt-1 font-mono text-gray-950 dark:text-white">{form.appPort}</dd>
					</div>
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Memory</dt>
						<dd class="mt-1 font-mono text-gray-950 dark:text-white">{form.memoryMb}</dd>
					</div>
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">CPU</dt>
						<dd class="mt-1 font-mono text-gray-950 dark:text-white">{form.cpuLimit}</dd>
					</div>
				</div>
				<div class="px-5 py-3">
					<dt class="text-xs text-gray-500 dark:text-gray-400">Profile</dt>
					<dd class="mt-1 text-gray-950 dark:text-white">{selectedProfile?.title ?? form.resourceProfile}</dd>
				</div>
				{#if form.deployMode !== 'static'}
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Database</dt>
						<dd class="mt-1 text-gray-950 dark:text-white">{form.sharedPostgres ? 'Shared PostgreSQL' : '-'}</dd>
					</div>
				{/if}
			</dl>
		</aside>
	</div>
</div>
