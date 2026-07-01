<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$api';
	import { toast } from '$stores/toast';

	let step = 1;
	let submitting = false;
	let error = '';
	let form = {
		name:        '',
		repoUrl:     '',
		branch:      'main',
		deployMode:  'auto' as 'auto' | 'dockerfile' | 'compose',
		mainService: '',
		appPort:     '3000',
		memoryMb:    '512',
		cpuLimit:    '0.5'
	};

	function next() { step = Math.min(step + 1, 3); }
	function back() { step = Math.max(step - 1, 1); }

	async function handleSubmit() {
		submitting = true;
		error = '';
		try {
			const project = await api.projects.create({
				name: form.name,
				repoUrl: form.repoUrl,
				branch: form.branch,
				deployMode: form.deployMode,
				mainService: form.mainService || null,
				appPort: Number(form.appPort),
				memoryLimitMb: Number(form.memoryMb),
				cpuLimit: Number(form.cpuLimit)
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

<div class="mx-auto max-w-2xl px-4 py-8 sm:px-6">
	<!-- Header -->
	<div class="mb-8">
		<a href="/projects" class="mb-4 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-900 dark:hover:text-white">
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
			</svg>
			Projects
		</a>
		<h1 class="text-xl font-bold text-gray-900 dark:text-white">New project</h1>
	</div>

	<!-- Step indicator -->
	<div class="mb-8 flex items-center gap-2">
		{#each [1, 2, 3] as s}
			<div class="flex items-center gap-2">
				<div class="flex h-7 w-7 items-center justify-center rounded-full text-xs font-semibold
							{step >= s ? 'bg-brand-600 text-white' : 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'}">
					{#if step > s}✓{:else}{s}{/if}
				</div>
				<span class="text-sm {step === s ? 'font-medium text-gray-900 dark:text-white' : 'text-gray-400'}">
					{['Repository', 'Configuration', 'Resources'][s - 1]}
				</span>
			</div>
			{#if s < 3}
				<div class="flex-1 border-t border-gray-200 dark:border-gray-800"></div>
			{/if}
		{/each}
	</div>

	<!-- Form card -->
	<div class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
		{#if error}
			<div class="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300">
				{error}
			</div>
		{/if}
		{#if step === 1}
			<h2 class="mb-4 font-semibold text-gray-900 dark:text-white">Repository</h2>
			<div class="space-y-4">
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="name">
						Project name
					</label>
					<input
						id="name"
						type="text"
						bind:value={form.name}
						placeholder="my-app"
						class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
							   dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
					/>
					<p class="mt-1 text-xs text-gray-500">Alphanumeric and dashes only. Will become your subdomain.</p>
				</div>
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="repo">
						Repository URL
					</label>
					<input
						id="repo"
						type="text"
						bind:value={form.repoUrl}
						placeholder="https://github.com/username/repo"
						class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
							   dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
					/>
				</div>
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="branch">
						Branch
					</label>
					<input
						id="branch"
						type="text"
						bind:value={form.branch}
						class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
							   dark:border-gray-700 dark:bg-gray-800 dark:text-white"
					/>
				</div>
			</div>

		{:else if step === 2}
			<h2 class="mb-4 font-semibold text-gray-900 dark:text-white">Configuration</h2>
			<div class="space-y-4">
				<div>
					<p class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
						Deploy mode
					</p>
					<div class="grid grid-cols-3 gap-2">
						{#each ['auto', 'dockerfile', 'compose'] as mode}
							<button
								type="button"
								on:click={() => (form.deployMode = mode as typeof form.deployMode)}
								class="rounded-lg border p-3 text-left text-sm transition-colors
									   {form.deployMode === mode
										? 'border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-900/20 dark:text-brand-400'
										: 'border-gray-200 text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800'}"
							>
								<div class="font-medium capitalize">{mode}</div>
								<div class="text-xs opacity-60">
									{mode === 'auto' ? 'Detect automatically' : mode === 'dockerfile' ? 'Single container' : 'Multi-service'}
								</div>
							</button>
						{/each}
					</div>
				</div>
				{#if form.deployMode === 'compose'}
					<div>
						<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="mainService">
							Main service name
						</label>
						<input
							id="mainService"
							type="text"
							bind:value={form.mainService}
							placeholder="app"
							class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
								   dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
						/>
					</div>
				{/if}
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="appPort">
						App port
					</label>
					<input
						id="appPort"
						type="number"
						bind:value={form.appPort}
						class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
							   dark:border-gray-700 dark:bg-gray-800 dark:text-white"
					/>
					<p class="mt-1 text-xs text-gray-500">Port your app listens on inside the container.</p>
				</div>
			</div>

		{:else}
			<h2 class="mb-4 font-semibold text-gray-900 dark:text-white">Resource limits</h2>
			<div class="space-y-4">
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="memory">
						Memory limit (MB)
					</label>
					<select
						id="memory"
						bind:value={form.memoryMb}
						class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
							   dark:border-gray-700 dark:bg-gray-800 dark:text-white"
					>
						<option value="256">256 MB</option>
						<option value="512">512 MB (default)</option>
						<option value="1024">1024 MB</option>
						<option value="2048">2048 MB</option>
					</select>
				</div>
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="cpu">
						CPU limit (cores)
					</label>
					<select
						id="cpu"
						bind:value={form.cpuLimit}
						class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
							   dark:border-gray-700 dark:bg-gray-800 dark:text-white"
					>
						<option value="0.25">0.25 cores</option>
						<option value="0.5">0.5 cores (default)</option>
						<option value="1">1 core</option>
						<option value="2">2 cores</option>
					</select>
				</div>
				<div class="rounded-lg bg-gray-50 p-3 text-xs text-gray-500 dark:bg-gray-800 dark:text-gray-400">
					Subdomain will be: <span class="font-mono font-medium text-gray-900 dark:text-white">{form.name || 'your-app'}.nabilrizkinavisa.me</span>
				</div>
			</div>
		{/if}
	</div>

	<!-- Nav buttons -->
	<div class="mt-4 flex justify-between">
		<button
			type="button"
			on:click={back}
			disabled={step === 1}
			class="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50
				   disabled:cursor-not-allowed disabled:opacity-40 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
		>
			Back
		</button>

		{#if step < 3}
			<button
				type="button"
				on:click={next}
				class="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
			>
				Next
			</button>
		{:else}
			<button
				type="button"
				on:click={handleSubmit}
				disabled={submitting}
				class="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
			>
				{submitting ? 'Creating...' : 'Create project'}
			</button>
		{/if}
	</div>
</div>
