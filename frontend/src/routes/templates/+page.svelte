<script lang="ts">
	import { Check, Database, Package, RefreshCw, ShieldCheck } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import { projectNameValidationMessage } from '$lib/validation/project';
	import { projectURL } from '$lib/utils/urls';
	import { appTemplates, appTemplateRepository, generateTemplateSecret, initialTemplateEnv, missingRequiredTemplateEnv, type AppTemplate } from '$lib/templates/app-templates';
	import { toast } from '$stores/toast';

	let selected: AppTemplate = appTemplates[0];
	let projectName = selected.id;
	let envValues: Record<string, string> = initialTemplateEnv(selected);
	let creating = false;
	let error = '';

	$: nameError = projectNameValidationMessage(projectName);
	$: publicURL = projectURL(projectName || selected.id, $page.url.protocol, $page.url.hostname);
	$: generatedSecretCount = selected.env.filter((field) => field.kind === 'secret').length;
	$: missingRequiredEnv = missingRequiredTemplateEnv(selected, envValues, publicURL);
	$: secondaryResourceEntries = Object.entries(selected.serviceResources ?? {});

	function chooseTemplate(template: AppTemplate) {
		selected = template;
		projectName = template.id;
		envValues = initialTemplateEnv(template);
		error = '';
	}

	function sourceLabel(template: AppTemplate) {
		switch (template.source.type) {
			case 'registry': return 'OCI image';
			case 'dockerfile': return 'Dockerfile';
			case 'compose': return 'Compose';
		}
	}

	function sourceDescription(template: AppTemplate) {
		switch (template.source.type) {
			case 'registry': return template.source.imageRef;
			case 'dockerfile': return `${template.source.repoUrl} · Dockerfile`;
			case 'compose': return `${template.source.mainService} via Docker Compose`;
		}
	}

	function fieldValue(key: string, kind: string) {
		return kind === 'public-url' ? publicURL : (envValues[key] ?? '');
	}

	function setEnvValue(key: string, value: string) {
		envValues = { ...envValues, [key]: value };
	}

	function regenerateSecret(key: string) {
		const field = selected.env.find((item) => item.key === key);
		if (!field || field.kind !== 'secret') return;
		setEnvValue(key, generateTemplateSecret(field));
	}

	function regenerateAllSecrets() {
		const next = { ...envValues };
		for (const field of selected.env) {
			if (field.kind === 'secret') next[field.key] = generateTemplateSecret(field);
		}
		envValues = next;
		toast.success('Generated fresh template secrets');
	}

	function environmentPayload() {
		return selected.env
			.map((field) => ({ key: field.key, value: field.kind === 'public-url' ? publicURL : (envValues[field.key] ?? '') }))
			.filter((item) => item.value.length > 0);
	}

	async function createTemplateProject() {
		if (creating) return;
		if (nameError) {
			error = nameError;
			return;
		}
		if (missingRequiredEnv.length > 0) {
			error = `Fill required environment values: ${missingRequiredEnv.join(', ')}`;
			return;
		}
		creating = true;
		error = '';
		try {
			const common = {
				name: projectName.trim().toLowerCase(),
				resourceProfile: 'custom',
				appPort: selected.appPort,
				memoryLimitMb: selected.memoryLimitMb,
				cpuLimit: selected.cpuLimit,
				serviceResources: selected.serviceResources ?? {},
				sharedPostgres: false,
				envVars: environmentPayload(),
				composeOverridePaths: [] as string[],
				composeProfiles: [] as string[],
				composeWorkdir: null as string | null,
				staticFrontendPath: null as string | null
			};

			let project;
			if (selected.source.type === 'registry') {
				project = await api.projects.create({
					...common,
					sourceType: 'registry',
					repoUrl: '',
					imageRef: selected.source.imageRef,
					branch: '',
					deployMode: 'image',
					mainService: null,
					composeFilePath: null,
					baseDirectory: null
				});
			} else if (selected.source.type === 'dockerfile') {
				project = await api.projects.create({
					...common,
					sourceType: 'git',
					repoUrl: selected.source.repoUrl,
					imageRef: null,
					branch: selected.source.branch,
					deployMode: 'dockerfile',
					mainService: null,
					composeFilePath: null,
					baseDirectory: selected.source.baseDirectory ?? null
				});
			} else {
				project = await api.projects.create({
					...common,
					sourceType: 'git',
					repoUrl: appTemplateRepository.repoUrl,
					imageRef: null,
					branch: appTemplateRepository.branch,
					deployMode: 'compose',
					mainService: selected.source.mainService,
					composeFilePath: selected.source.composeFilePath,
					baseDirectory: selected.source.baseDirectory
				});
			}

			toast.success(`${selected.name} project created`);
			await goto(`/projects/${project.id}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create project from template';
			toast.error(error);
		} finally {
			creating = false;
		}
	}
</script>

<svelte:head>
	<title>App templates · MyPaas</title>
</svelte:head>

<div class="page-shell space-y-5 py-6">
	<div>
		<h1 class="text-xl font-semibold text-gray-950 dark:text-white">App templates</h1>
		<p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-gray-400">Create real OSS workloads using the existing MyPaas project and deployment engine. Templates only provide safe defaults, generated secrets, and known compatibility boundaries.</p>
	</div>

	<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
		{#each appTemplates as template}
			<button type="button" class={`surface p-4 text-left transition-colors ${selected.id === template.id ? 'ring-2 ring-gray-950 dark:ring-white' : 'hover:bg-gray-50 dark:hover:bg-neutral-900'}`} on:click={() => chooseTemplate(template)}>
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0">
						<p class="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">{template.category}</p>
						<h2 class="mt-1 text-base font-semibold text-gray-950 dark:text-white">{template.name}</h2>
					</div>
					{#if selected.id === template.id}<Check class="h-4 w-4 shrink-0 text-gray-950 dark:text-white" aria-hidden="true" />{/if}
				</div>
				<p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">{template.description}</p>
				<div class="mt-3 flex flex-wrap gap-2 text-[11px] text-gray-500 dark:text-gray-400">
					<span class="font-medium text-gray-700 dark:text-gray-300">Catalogued pattern</span>
					<span>·</span><span>{sourceLabel(template)}</span>
					<span>·</span><span>:{template.appPort}</span>
					{#if template.persistent}<span>·</span><span>Persistent</span>{/if}
				</div>
			</button>
		{/each}
	</div>

	<div class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_22rem]">
		<SectionPanel title={`Configure ${selected.name}`} description="Review the platform-owned defaults before creating the project.">
			<div class="space-y-5">
				{#if error}<div class="alert-danger">{error}</div>{/if}
				<div>
					<label class="field-label" for="template-project-name">Project name</label>
					<input id="template-project-name" class="field w-full max-w-xl" bind:value={projectName} aria-invalid={nameError ? 'true' : undefined} />
					{#if nameError}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{nameError}</p>{:else}<p class="mt-1 font-mono text-[11px] text-gray-500 dark:text-gray-400">{publicURL}</p>{/if}
				</div>

				{#if selected.env.length > 0}
					<div>
						<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
							<div><p class="text-sm font-medium text-gray-950 dark:text-white">Environment</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Secrets are generated locally and then stored through MyPaas encrypted env storage. Required values must be present before project creation.</p></div>
							{#if generatedSecretCount > 0}<ActionButton type="button" variant="secondary" size="xs" on:click={regenerateAllSecrets}><RefreshCw slot="icon" class="h-3.5 w-3.5" />Regenerate secrets</ActionButton>{/if}
						</div>
						<div class="divide-y divide-gray-100 border-y border-gray-100 dark:divide-neutral-800 dark:border-neutral-800">
							{#each selected.env as field}
								<div class="grid gap-2 py-3 lg:grid-cols-[minmax(10rem,0.8fr)_minmax(14rem,1.5fr)_auto] lg:items-center">
									<div><p class="font-mono text-sm font-medium text-gray-950 dark:text-white">{field.key}</p><p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">{field.label}{field.required ? ' · required' : ''}</p></div>
									<input class="field w-full font-mono" type={field.kind === 'secret' ? 'password' : 'text'} value={fieldValue(field.key, field.kind)} disabled={field.kind === 'public-url'} aria-invalid={field.required && missingRequiredEnv.includes(field.key) ? 'true' : undefined} on:input={(event) => setEnvValue(field.key, (event.currentTarget as HTMLInputElement).value)} />
									{#if field.kind === 'secret'}<ActionButton type="button" variant="secondary" size="xs" on:click={() => regenerateSecret(field.key)}>Generate</ActionButton>{:else}<span class="text-xs text-gray-500 dark:text-gray-400">{field.kind === 'public-url' ? 'Managed' : field.required ? 'Required' : 'Optional'}</span>{/if}
									<p class="text-xs leading-5 text-gray-500 dark:text-gray-400 lg:col-start-2 lg:col-span-2">{field.description}</p>
								</div>
							{/each}
						</div>
						{#if missingRequiredEnv.length > 0}<p class="mt-2 text-xs text-red-600 dark:text-red-300">Required before create: {missingRequiredEnv.join(', ')}</p>{/if}
					</div>
				{/if}

				<ActionButton variant="primary" loading={creating} loadingLabel="Creating" disabled={Boolean(nameError) || missingRequiredEnv.length > 0} on:click={createTemplateProject}>
					<Package slot="icon" class="h-4 w-4" />
					Create {selected.name}
				</ActionButton>
			</div>
		</SectionPanel>

		<div class="space-y-4">
			<SectionPanel title="Compatibility status" description="Curated platform metadata, not a capacity or production-readiness claim.">
				<div class="flex gap-2">
					<ShieldCheck class="mt-0.5 h-4 w-4 shrink-0" />
					<div>
						<p class="text-sm font-medium text-gray-950 dark:text-white">Catalogued pattern</p>
						<p class="mt-1 font-mono text-[11px] text-gray-500 dark:text-gray-400">compatibility/{selected.compatibility.catalogId}</p>
					</div>
				</div>
				<p class="mt-3 text-xs leading-5 text-gray-500 dark:text-gray-400">This template maps to a declared MyPaas compatibility workload pattern. Live PASS/FAIL evidence belongs to compatibility runs; the dashboard does not invent a throughput, user-count, or hardware-capacity result.</p>
			</SectionPanel>

			<SectionPanel title="Deployment contract" description="What this template asks MyPaas to own.">
				<div class="space-y-3 text-sm text-gray-600 dark:text-gray-300">
					<div class="flex gap-2"><Package class="mt-0.5 h-4 w-4 shrink-0" /><span>{sourceDescription(selected)}</span></div>
					<div class="flex gap-2"><Database class="mt-0.5 h-4 w-4 shrink-0" /><span>{selected.persistent ? 'Persistent Docker-managed storage is expected.' : 'No persistent storage is required by the baseline template.'}</span></div>
					<div class="flex gap-2"><ShieldCheck class="mt-0.5 h-4 w-4 shrink-0" /><span>{selected.memoryLimitMb} MB memory · {selected.cpuLimit} CPU main-service guardrail.</span></div>
					{#if secondaryResourceEntries.length > 0}
						<div class="border-t border-gray-100 pt-3 dark:border-neutral-800">
							<p class="text-xs font-medium text-gray-700 dark:text-gray-300">Secondary service guardrails</p>
							{#each secondaryResourceEntries as [service, resource]}
								<p class="mt-1 font-mono text-[11px] text-gray-500 dark:text-gray-400">{service}: {resource.memoryLimitMb} MB · {resource.cpuLimit} CPU</p>
							{/each}
						</div>
					{/if}
				</div>
			</SectionPanel>

			{#if selected.limitations.length > 0}
				<SectionPanel title="Known boundaries" description="Compatibility limits are shown instead of hidden behind a generic readiness claim.">
					<ul class="space-y-2 text-sm text-gray-600 dark:text-gray-300">{#each selected.limitations as limitation}<li>• {limitation}</li>{/each}</ul>
				</SectionPanel>
			{/if}
		</div>
	</div>
</div>
