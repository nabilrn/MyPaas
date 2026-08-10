<script lang="ts">
	import { Check, Copy, Upload, X } from '@lucide/svelte';
	import { onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import Breadcrumbs from '$components/Breadcrumbs.svelte';
	import IconButton from '$components/IconButton.svelte';
	import PageHeader from '$components/PageHeader.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import SegmentedChoice from '$components/SegmentedChoice.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import { projectHost, projectURL } from '$lib/utils/urls';
	import { projectCreationReadiness, resolveProjectAppPort, suggestProjectName } from '$lib/validation/project';
	import type { ComposeCandidate, ComposeIssue, ComposePlan, ComposePortPlan, ComposeServicePlan, DeployModeDetection, EnvVarDiscovery, RepoInspection, RepoTreeEntry, ResourceProfile } from '$types';

	type SourceType = 'git' | 'registry';
	type DeployModeChoice = 'auto' | 'dockerfile' | 'compose' | 'static' | 'image';
	type EnvDraft = EnvVarDiscovery & { value: string };
	type PortSource = 'unresolved' | 'detected' | 'manual' | 'static';
	type ComposeServicePlanPayload = Omit<ComposeServicePlan, 'ports' | 'expose' | 'dependsOn'> & {
		ports?: ComposePortPlan[] | null;
		expose?: number[] | null;
		dependsOn?: string[] | null;
	};
	type ComposePlanPayload = Omit<ComposePlan, 'requiredEnvVars' | 'services' | 'issues'> & {
		requiredEnvVars?: string[] | null;
		services?: ComposeServicePlanPayload[] | null;
		issues?: ComposeIssue[] | null;
	};

	const publicOriginEnvKeys = new Set([
		'ALLOWED_ORIGINS',
		'APP_ORIGIN',
		'APP_URL',
		'CLIENT_URL',
		'CORS_ORIGIN',
		'CORS_ORIGINS',
		'FRONTEND_URL',
		'PUBLIC_APP_ORIGIN',
		'PUBLIC_ORIGIN',
		'PUBLIC_URL'
	]);
	const breadcrumbs = [
		{ label: 'Projects', href: '/projects' },
		{ label: 'New project' }
	];

	let submitting = false;
	let detecting = false;
	let inspectingRepo = false;
	let error = '';
	let detectMessage = '';
	let repoInspectError = '';
	let repoInspectMessage = '';
	let repoInspectTimer: ReturnType<typeof setTimeout> | undefined = undefined;
	let repoInspectRequest = 0;
	let lastRepoInspectKey = '';
	let branchOptions: string[] = [];
	let defaultBranch = '';
	let repoTree: RepoTreeEntry[] = [];
	let repoTreeTruncated = false;
	let composePlan: ComposePlan | null = null;
	let detectedServices: string[] = [];
	let composeCandidates: ComposeCandidate[] = [];
	let composeCandidatesLoading = false;
	let composeCandidatesError = '';
	let envDrafts: EnvDraft[] = [];
	let newEnvKey = '';
	let appPortSource: PortSource = 'unresolved';
	let projectNameTouched = false;
	let deployModeManual = false;
	let envFileInput: HTMLInputElement | null = null;
	let copiedHandoffPrompt = '';
	let handoffCopyTimer: ReturnType<typeof setTimeout> | undefined;
	let form = {
		name: '',
		sourceType: 'git' as SourceType,
		repoUrl: '',
		imageRef: '',
		branch: '',
		deployMode: 'auto' as DeployModeChoice,
		mainService: '',
		appPort: '',
		resourceProfile: 'node-python' as ResourceProfile,
		memoryMb: '256',
		cpuLimit: '0.35',
		sharedPostgres: false,
		composeFilePath: '',
		composeOverridePaths: '',
		composeProfiles: '',
		composeWorkdir: '',
		staticFrontendPath: '',
		baseDirectory: ''
	};
	let staticFrontendCandidates: string[] = [];

	const sourceTypeOptions = [
		{ value: 'git', label: 'Git Repository', description: 'Clone & build' },
		{ value: 'registry', label: 'Container Registry', description: 'Pull image' }
	];

	const deployModes: Array<{ id: DeployModeChoice; title: string; body: string }> = [
		{ id: 'auto', title: 'Auto', body: 'Detect' },
		{ id: 'dockerfile', title: 'Dockerfile', body: 'Single app' },
		{ id: 'compose', title: 'Compose', body: 'Multi-service' },
		{ id: 'static', title: 'Static', body: 'File server' }
	];
	const resourceProfiles: Array<{ id: ResourceProfile; title: string; memoryMb: string; cpuLimit: string }> = [
		{ id: 'node-python', title: 'Node/Python', memoryMb: '256', cpuLimit: '0.35' },
		{ id: 'go-small', title: 'Go small', memoryMb: '128', cpuLimit: '0.2' },
		{ id: 'compose-main', title: 'Compose main', memoryMb: '256', cpuLimit: '0.35' },
		{ id: 'static', title: 'Static/no-runtime', memoryMb: '64', cpuLimit: '0.1' },
		{ id: 'custom', title: 'Custom', memoryMb: '512', cpuLimit: '0.5' }
	];

	$: previewHost = projectHost(form.name || 'your-app', $page.url.hostname);
	$: previewOrigin = projectURL(form.name || 'your-app', $page.url.protocol, $page.url.hostname);
	$: selectedProfile = resourceProfiles.find((profile) => profile.id === form.resourceProfile);
	$: managedDatabaseUrl = form.sharedPostgres && form.deployMode !== 'static';
	$: effectiveAppPort = form.deployMode === 'static' ? '80' : form.appPort || 'Not detected';
	$: handoffEnvKeys = Array.from(new Set([
		...envDrafts.map((item) => normalizeEnvKey(item.key)).filter(Boolean),
		...(managedDatabaseUrl ? ['DATABASE_URL'] : [])
	])).sort();
	$: handoffPrompt = buildDeploymentHandoffPrompt(
		form.deployMode,
		form.name,
		form.repoUrl,
		form.branch,
		form.mainService,
		form.appPort,
		appPortSource,
		handoffEnvKeys
	);
	$: deployModeOptions = deployModes.map((mode) => ({
		value: mode.id,
		label: mode.title,
		description: mode.body
	}));
	$: portStateLabel = form.deployMode === 'static'
		? 'Managed by Caddy'
		: appPortSource === 'detected'
			? 'Detected from repository'
			: appPortSource === 'manual'
				? 'Advanced manual override'
				: 'Not detected yet';
	$: composeBlockingIssues = composePlan?.issues.filter((issue) => issue.severity === 'error') ?? [];
	$: envDraftValueByKey = new Map(
		envDrafts
			.map((item) => [normalizeEnvKey(item.key), item.value] as const)
			.filter(([key]) => Boolean(key))
	);
	$: normalizedComposeRequiredEnvKeys = Array.from(
		new Set((composePlan?.requiredEnvVars ?? []).map(normalizeEnvKey).filter(Boolean))
	);
	$: missingRequiredEnvKeys = normalizedComposeRequiredEnvKeys
		.filter((key) => !(managedDatabaseUrl && key === 'DATABASE_URL'))
		.filter((key) => !((envDraftValueByKey.get(key)?.trim()?.length ?? 0) > 0));
	$: composeDisabledReason = composeBlockingIssues[0]?.message
		?? (missingRequiredEnvKeys.length > 0 ? `Fill required env values: ${missingRequiredEnvKeys.slice(0, 3).join(', ')}${missingRequiredEnvKeys.length > 3 ? '...' : ''}` : '');
	$: portToServiceMap = buildPortToServiceMap(composePlan?.services ?? []);
	$: localhostEnvWarnings = detectLocalhostInEnvDrafts(envDrafts, portToServiceMap);
	$: currentRepoInspectKey = [form.repoUrl.trim(), form.branch.trim(), form.baseDirectory.trim()].join('\n');
	$: repositoryInspectionCurrent = Boolean(
		form.repoUrl.trim()
		&& form.branch.trim()
		&& lastRepoInspectKey === currentRepoInspectKey
		&& !repoInspectError
	);
	$: sourceReady = form.sourceType === 'registry'
		? Boolean(form.imageRef.trim())
		: Boolean(form.repoUrl.trim() && form.branch.trim() && repositoryInspectionCurrent);
	$: creationReadiness = projectCreationReadiness({
		name: form.name,
		sourceType: form.sourceType,
		sourceReady,
		deployMode: form.deployMode,
		appPort: form.appPort,
		composeDisabledReason,
		busy: submitting || detecting || inspectingRepo
	});
	$: canSubmit = creationReadiness.ready;
	$: createDisabledReason = form.sourceType === 'git' && !form.repoUrl.trim()
		? 'Repository URL is required'
		: form.sourceType === 'git' && !form.branch.trim() && form.repoUrl.trim()
			? 'Select a branch after repository validation'
			: form.sourceType === 'git' && repoInspectError
				? repoInspectError
				: creationReadiness.reason;
	$: reviewStateLabel = creationReadiness.state;
	$: detectionStateLabel = detecting
		? 'Inspecting runtime'
		: inspectingRepo
			? 'Loading repository'
		: detectMessage
			? detectMessage
			: repoInspectMessage
				? repoInspectMessage
			: form.repoUrl.trim()
				? form.branch.trim()
					? 'Ready for automatic analysis'
					: 'Select a branch'
				: 'Waiting for repository URL';
	$: detectionStateBody = detecting
		? 'MyPaas is checking the selected branch for Dockerfile, Compose, static assets, ports, services, and env hints.'
		: inspectingRepo
			? 'Fetching branches and the repository structure for the selected base directory.'
		: detectMessage
			? detectedServices.length > 0
				? `Services: ${detectedServices.join(', ')}`
				: 'Runtime and defaults have been applied from the repository.'
			: repoInspectError
				? repoInspectError
			: form.repoUrl.trim()
				? form.branch.trim()
					? 'MyPaas analyzes runtime, container port, services, and environment defaults automatically.'
					: 'Branches load automatically after the repository URL is entered.'
				: 'Paste a repository URL before running detection.';

	function repositoryInspectionKey() {
		return `${form.repoUrl.trim()}\n${form.branch.trim()}\n${form.baseDirectory.trim()}`;
	}

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

	function chooseSourceType(sourceType: SourceType) {
		if (form.sourceType === sourceType) return;
		form.sourceType = sourceType;
		deployModeManual = false;
		error = '';
		detectMessage = '';
		composePlan = null;
		if (sourceType === 'registry') {
			resetRepositoryInspection();
			form.deployMode = 'image';
			form.mainService = '';
			form.composeFilePath = '';
			form.composeOverridePaths = '';
			form.composeProfiles = '';
			form.composeWorkdir = '';
			form.staticFrontendPath = '';
			form.appPort = '';
			appPortSource = 'unresolved';
			if (form.resourceProfile !== 'custom') applyResourceProfile('node-python');
		} else if (form.deployMode === 'image') {
			form.deployMode = 'auto';
			form.appPort = '';
			appPortSource = 'unresolved';
			if (form.repoUrl.trim()) scheduleRepositoryInspection();
		}
	}

	function chooseDeployMode(mode: DeployModeChoice, manual = true) {
		deployModeManual = manual && mode !== 'auto';
		form.deployMode = mode;
		if (mode !== 'compose') {
			composePlan = null;
			composeCandidates = [];
			composeCandidatesError = '';
		}
		if (mode === 'static') {
			form.appPort = '80';
			appPortSource = 'static';
			form.mainService = '';
			form.sharedPostgres = false;
		} else if (appPortSource === 'static') {
			form.appPort = '';
			appPortSource = 'unresolved';
		} else if (!form.appPort) {
			appPortSource = 'unresolved';
		}
		if (form.resourceProfile !== 'custom') {
			applyResourceProfile(defaultProfileForMode(mode));
		}
	}

	function applyDetectedMode(detected: DeployModeDetection) {
		const manualPort = appPortSource === 'manual' ? form.appPort : '';
		if (detected.branch) {
			form.branch = detected.branch;
		}
		defaultBranch = detected.defaultBranch || defaultBranch;
		branchOptions = normalizeBranches(detected.branches, detected.branch || defaultBranch);
		repoTree = detected.tree ?? repoTree;
		repoTreeTruncated = detected.treeTruncated ?? repoTreeTruncated;
		composePlan = normalizeComposePlan(detected.composePlan);
		composeCandidates = Array.isArray(detected.composeCandidates) ? detected.composeCandidates : [];
		staticFrontendCandidates = Array.isArray(detected.staticFrontendCandidates) ? detected.staticFrontendCandidates : [];
		chooseDeployMode(detected.deployMode, false);
		if (detected.mainService) {
			form.mainService = detected.mainService;
		}
		if (staticFrontendCandidates.length > 0 && !form.staticFrontendPath) {
			form.staticFrontendPath = staticFrontendCandidates[0];
		}
		if (detected.composeFile && !form.composeFilePath) {
			form.composeFilePath = detected.composeFile;
		}
		if (detected.deployMode === 'static') {
			form.appPort = '80';
			appPortSource = 'static';
		} else if (detected.appPort > 0) {
			form.appPort = String(detected.appPort);
			appPortSource = 'detected';
		} else if (manualPort) {
			form.appPort = manualPort;
			appPortSource = 'manual';
		} else {
			form.appPort = '';
			appPortSource = 'unresolved';
		}
		detectedServices = detected.services ?? [];
		mergeDiscoveredEnvVars(detected.envVars ?? []);
		const branchSuffix = detected.branch ? ` on ${detected.branch}` : '';
		detectMessage = detected.deployMode === 'compose'
			? `Compose${detected.composeFile ? `: ${detected.composeFile}` : ''}`
			: detected.deployMode === 'static'
				? 'Static site'
				: 'Dockerfile';
		detectMessage += branchSuffix;
	}

	async function refreshComposeCandidates(showToast = false): Promise<void> {
		if (form.deployMode !== 'compose') return;
		const repoUrl = form.repoUrl.trim();
		const branch = form.branch.trim();
		if (!repoUrl || !branch) return;
		composeCandidatesLoading = true;
		composeCandidatesError = '';
		try {
			const result = await api.projects.detectCompose({ repoUrl, branch, baseDirectory: form.baseDirectory.trim() || undefined });
			composeCandidates = Array.isArray(result.candidates) ? result.candidates : [];
			if (composeCandidates.length > 0 && !form.composeFilePath) {
				form.composeFilePath = composeCandidates[0].path;
			}
			if (showToast) {
				toast.success(`Found ${composeCandidates.length} compose candidate${composeCandidates.length === 1 ? '' : 's'}`);
			}
		} catch (err) {
			composeCandidates = [];
			composeCandidatesError = err instanceof Error ? err.message : 'Failed to scan for compose files';
			if (showToast) {
				toast.error(composeCandidatesError);
			}
		} finally {
			composeCandidatesLoading = false;
		}
	}

	function selectComposeCandidate(path: string) {
		form.composeFilePath = path;
		form.composeWorkdir = '';
	}

	function normalizeComposePlan(plan: ComposePlan | null | undefined): ComposePlan | null {
		if (!plan) return null;
		const payload = plan as ComposePlanPayload;
		return {
			...plan,
			requiredEnvVars: Array.isArray(payload.requiredEnvVars) ? payload.requiredEnvVars : [],
			services: Array.isArray(payload.services)
				? payload.services.map((service) => ({
					...service,
					ports: Array.isArray(service.ports) ? service.ports : [],
					expose: Array.isArray(service.expose) ? service.expose : [],
					dependsOn: Array.isArray(service.dependsOn) ? service.dependsOn : []
				}))
				: [],
			issues: Array.isArray(payload.issues) ? payload.issues : []
		};
	}

	function formatComposeServicePorts(service: ComposeServicePlan) {
		const ports = Array.isArray(service.ports) ? service.ports : [];
		const expose = Array.isArray(service.expose) ? service.expose : [];
		if (ports.length > 0) {
			return ports.map((port) => `${port.published ? `${port.published}:` : ''}${port.target}`).join(', ');
		}
		return expose.length > 0 ? expose.join(', ') : '-';
	}

	function normalizeBranches(branches: string[] | undefined, selected = '') {
		const seen = new Set<string>();
		const out: string[] = [];
		const add = (branch: string) => {
			branch = branch.trim();
			if (!branch || seen.has(branch)) return;
			seen.add(branch);
			out.push(branch);
		};
		for (const branch of branches ?? []) {
			add(branch);
		}
		add(selected);
		return out;
	}

	function clearDetectedSourceState() {
		repoInspectRequest += 1;
		error = '';
		detectMessage = '';
		repoInspectError = '';
		repoInspectMessage = '';
		repoTree = [];
		repoTreeTruncated = false;
		composePlan = null;
		composeCandidates = [];
		composeCandidatesError = '';
		staticFrontendCandidates = [];
		form.composeFilePath = '';
		form.composeWorkdir = '';
		form.staticFrontendPath = '';
		detectedServices = [];
		lastRepoInspectKey = '';
		if (!deployModeManual && form.sourceType === 'git') {
			form.deployMode = 'auto';
			form.mainService = '';
			if (appPortSource !== 'manual') {
				form.appPort = '';
				appPortSource = 'unresolved';
			}
		}
	}

	function handleNameInput(event: Event) {
		projectNameTouched = true;
		form.name = (event.currentTarget as HTMLInputElement).value;
	}

	function handleRepoUrlInput(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value;
		if (value === form.repoUrl) return;
		form.repoUrl = value;
		form.branch = '';
		deployModeManual = false;
		if (!projectNameTouched) form.name = suggestProjectName(value);
		resetRepositoryInspection();
		scheduleRepositoryInspection();
	}

	function handleImageRefInput(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value;
		form.imageRef = value;
		if (!projectNameTouched) form.name = suggestProjectName(value);
	}

	function handleBaseDirectoryInput(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value;
		if (value === form.baseDirectory) return;
		form.baseDirectory = value;
		clearDetectedSourceState();
		scheduleRepositoryInspection();
	}

	function resetRepositoryInspection() {
		clearDetectedSourceState();
		branchOptions = [];
		defaultBranch = '';
	}

	function scheduleRepositoryInspection() {
		if (repoInspectTimer) {
			clearTimeout(repoInspectTimer);
		}
		if (!form.repoUrl.trim()) return;
		repoInspectTimer = setTimeout(() => {
			void inspectRepository().catch(() => undefined);
		}, 700);
	}

	function handleBranchChange(event: Event) {
		form.branch = (event.currentTarget as HTMLSelectElement).value;
		clearDetectedSourceState();
		void inspectRepository(false, true).catch(() => undefined);
	}

	async function inspectRepository(showToast = false, force = false): Promise<RepoInspection | undefined> {
		const repoUrl = form.repoUrl.trim();
		if (!repoUrl) return undefined;
		if (repoInspectTimer) {
			clearTimeout(repoInspectTimer);
			repoInspectTimer = undefined;
		}

		const requestedBranch = form.branch.trim();
		const requestKey = `${repoUrl}\n${requestedBranch}\n${form.baseDirectory.trim()}`;
		if (!force && requestKey === lastRepoInspectKey && !repoInspectError) {
			return undefined;
		}

		const requestId = ++repoInspectRequest;
		inspectingRepo = true;
		repoInspectError = '';
		try {
			const inspection = await api.projects.inspectRepository({
				repoUrl,
				branch: requestedBranch,
				baseDirectory: form.baseDirectory.trim() || undefined
			});
			if (requestId !== repoInspectRequest) {
				return undefined;
			}
			defaultBranch = inspection.defaultBranch || inspection.branch;
			if (!form.branch.trim() && inspection.branch) {
				form.branch = inspection.branch;
			}
			branchOptions = normalizeBranches(inspection.branches, form.branch || inspection.branch || defaultBranch);
			repoTree = inspection.tree ?? [];
			repoTreeTruncated = inspection.treeTruncated ?? false;
			repoInspectMessage = branchOptions.length === 1
				? '1 branch available'
				: `${branchOptions.length} branches available`;
			lastRepoInspectKey = repositoryInspectionKey();
			if (showToast) {
				toast.success('Repository validated');
			}
			if (!deployModeManual) {
				setTimeout(() => {
					if (detecting || deployModeManual || lastRepoInspectKey !== repositoryInspectionKey()) return;
					void handleDetectMode(false).catch(() => undefined);
				}, 0);
			}
			return inspection;
		} catch (err) {
			if (requestId !== repoInspectRequest) {
				return undefined;
			}
			const message = err instanceof Error ? err.message : 'Failed to inspect repository';
			repoInspectError = message;
			repoInspectMessage = '';
			repoTree = [];
			repoTreeTruncated = false;
			lastRepoInspectKey = '';
			if (showToast) {
				toast.error(message);
			}
			throw err;
		} finally {
			if (requestId === repoInspectRequest) {
				inspectingRepo = false;
			}
		}
	}

	function markCustomProfile() {
		form.resourceProfile = 'custom';
	}

	function mergeDiscoveredEnvVars(vars: EnvVarDiscovery[]) {
		const existing = new Set(envDrafts.map((item) => normalizeEnvKey(item.key)));
		const nextDrafts = envDrafts.map((item) => {
			const discovered = vars.find((candidate) => normalizeEnvKey(candidate.key) === normalizeEnvKey(item.key));
			if (!discovered) return item;
			const defaultValue = discoveredEnvDefaultValue(discovered);
			return {
				...item,
				source: mergeEnvSources(item.source, discovered.source),
				sensitive: item.sensitive || discovered.sensitive,
				defaultValue: item.defaultValue ?? discovered.defaultValue,
				services: discovered.services ?? item.services,
				conflict: discovered.conflict ?? item.conflict,
				value: item.value || defaultValue
			};
		});
		for (const item of vars) {
			const key = normalizeEnvKey(item.key);
			if (!key || existing.has(key)) continue;
			nextDrafts.push({ ...item, key, value: discoveredEnvDefaultValue({ ...item, key }) });
			existing.add(key);
		}
		envDrafts = nextDrafts.sort((a, b) => a.key.localeCompare(b.key));
	}

	function discoveredEnvDefaultValue(item: EnvVarDiscovery) {
		if (item.sensitive) return '';
		return item.defaultValue ?? inferredProjectEnvValue(item.key);
	}

	function inferredProjectEnvValue(key: string) {
		return publicOriginEnvKeys.has(normalizeEnvKey(key)) ? previewOrigin : '';
	}

	function mergeEnvSources(current: string, discovered: string) {
		if (!current) return discovered;
		if (!discovered || current.split(', ').includes(discovered)) return current;
		return `${current}, ${discovered}`;
	}

	function addEnvVar() {
		const key = normalizeEnvKey(newEnvKey);
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

	function updateEnvDraftKey(index: number, value: string) {
		const key = normalizeEnvKey(value);
		envDrafts = envDrafts.map((item, itemIndex) => itemIndex === index
			? { ...item, key, sensitive: item.sensitive || isSensitiveEnvKey(key) }
			: item
		);
	}

	function updateEnvDraftValue(index: number, value: string) {
		envDrafts = envDrafts.map((item, itemIndex) => itemIndex === index ? { ...item, value } : item);
	}

	function triggerEnvFileImport() {
		envFileInput?.click();
	}

	async function handleEnvFileImport(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;

		try {
			const parsed = parseEnvFile(await file.text());
			if (parsed.vars.length === 0) {
				toast.error('No valid env variables found');
				return;
			}
			mergeEnvFileVars(parsed.vars);
			const skippedSuffix = parsed.skipped > 0 ? `, skipped ${parsed.skipped}` : '';
			toast.success(`Imported ${parsed.vars.length} env variable${parsed.vars.length === 1 ? '' : 's'}${skippedSuffix}`);
		} catch {
			toast.error('Failed to import env file');
		} finally {
			input.value = '';
		}
	}

	function parseEnvFile(content: string): { vars: EnvDraft[]; skipped: number } {
		const vars: EnvDraft[] = [];
		let skipped = 0;
		for (const rawLine of content.replace(/^\uFEFF/, '').replace(/\r\n/g, '\n').split('\n')) {
			let line = rawLine.trim();
			if (!line || line.startsWith('#')) continue;
			if (line.startsWith('export ')) {
				line = line.slice('export '.length).trim();
			}
			const separatorIndex = line.indexOf('=');
			if (separatorIndex <= 0) {
				skipped++;
				continue;
			}
			const key = normalizeEnvKey(line.slice(0, separatorIndex));
			if (!key) {
				skipped++;
				continue;
			}
			vars.push({
				key,
				value: unwrapEnvValue(stripEnvInlineComment(line.slice(separatorIndex + 1).trim()).trim()),
				source: 'env-file',
				sensitive: isSensitiveEnvKey(key)
			});
		}
		return { vars, skipped };
	}

	function stripEnvInlineComment(value: string) {
		let quote = '';
		let escaped = false;
		for (let index = 0; index < value.length; index += 1) {
			const char = value[index];
			if (escaped) {
				escaped = false;
				continue;
			}
			if (quote === '"' && char === '\\') {
				escaped = true;
				continue;
			}
			if (!quote && (char === '"' || char === "'")) {
				quote = char;
				continue;
			}
			if (quote === char) {
				quote = '';
				continue;
			}
			if (!quote && char === '#' && (index === 0 || /\s/.test(value[index - 1]))) {
				return value.slice(0, index).trimEnd();
			}
		}
		return value;
	}

	function unwrapEnvValue(value: string) {
		if (value.length < 2) return value;
		const quote = value[0];
		if ((quote !== '"' && quote !== "'") || value[value.length - 1] !== quote) {
			return value;
		}
		const inner = value.slice(1, -1);
		if (quote === "'") return inner;
		return inner.replace(/\\n/g, '\n').replace(/\\r/g, '\r').replace(/\\t/g, '\t').replace(/\\"/g, '"').replace(/\\\\/g, '\\');
	}

	function mergeEnvFileVars(vars: EnvDraft[]) {
		const incoming = new Map<string, EnvDraft>();
		for (const item of vars) {
			const key = normalizeEnvKey(item.key);
			if (key) {
				incoming.set(key, { ...item, key });
			}
		}
		const nextDrafts = envDrafts.map((item) => {
			const key = normalizeEnvKey(item.key);
			const imported = incoming.get(key);
			if (!imported) return item;
			incoming.delete(key);
			return {
				...item,
				key,
				value: imported.value,
				source: imported.source,
				sensitive: item.sensitive || imported.sensitive
			};
		});
		envDrafts = [...nextDrafts, ...incoming.values()].sort((a, b) => a.key.localeCompare(b.key));
	}

	function isSensitiveEnvKey(key: string) {
		return /SECRET|TOKEN|PASSWORD|PASS|KEY|DATABASE_URL|DSN|PRIVATE/i.test(key);
	}

	function normalizeEnvKey(value: string) {
		return value.trim().toUpperCase().replace(/[^A-Z0-9_]/g, '_');
	}

	function handleNewEnvKeydown(event: KeyboardEvent) {
		if (event.key !== 'Enter') return;
		event.preventDefault();
		addEnvVar();
	}

	function handleAppPortInput(event: Event) {
		form.appPort = (event.currentTarget as HTMLInputElement).value;
		appPortSource = form.appPort ? 'manual' : 'unresolved';
	}

	function buildDeploymentHandoffPrompt(
		mode: DeployModeChoice,
		projectName: string,
		repoUrl: string,
		branch: string,
		mainService: string,
		appPort: string,
		portSource: PortSource,
		envKeys: string[]
	) {
		if (mode !== 'dockerfile' && mode !== 'compose') return '';

		const portContext = portSource === 'detected'
			? 'detected by MyPaas'
			: portSource === 'manual'
				? 'set as an Advanced override'
				: 'not resolved yet';
		const portRequirement = appPort.trim()
			? `- The container port is ${appPort.trim()} (${portContext}). Keep the application listening on 0.0.0.0:${appPort.trim()} and declare/expose that container port. MyPaas manages the host port and Caddy route automatically.`
			: '- Make the HTTP application listen on one explicit container port, bind it to 0.0.0.0, and declare that port with Dockerfile EXPOSE or Compose expose/ports so MyPaas can detect it. Do not choose a host port; MyPaas manages host allocation and Caddy routing.';
		const envRequirement = envKeys.length > 0
			? `- Preserve and document these environment keys already discovered by MyPaas: ${envKeys.join(', ')}. PORT or APP_PORT are valid when the application uses them to select its container listening port. Do not put secret values in Git.`
			: '- Discover the runtime environment keys the application needs. Add keys only to .env.example with safe placeholders. PORT or APP_PORT are valid when required by the application. Never put secret values in Git.';
		const repositoryContext = [
			projectName.trim() ? `- MyPaas project: ${projectName.trim()}` : '',
			repoUrl.trim() ? `- Repository: ${repoUrl.trim()}` : '',
			branch.trim() ? `- Branch: ${branch.trim()}` : ''
		].filter(Boolean);
		const modeRequirements = mode === 'dockerfile'
			? [
				'- Create or repair a root-level Dockerfile and .dockerignore. Focus on the Dockerfile deployment contract; do not add Compose only for this deployment.',
				'- Reuse the project\'s existing package manager, lockfile, build command, and production start command. Use a multi-stage build when it materially reduces the runtime image.',
				'- The final container must run the application in the foreground and start without an interactive shell.',
				portRequirement,
				'- Run as a non-root user when the framework and filesystem requirements allow it.'
			]
			: [
				'- Create or repair a root-level compose.yml that includes at least one **buildable application service** (with a `build` context and Dockerfile) serving HTTP. A compose file containing only infrastructure services (database, cache, broker) without an app is incomplete and will fail deployment.',
				mainService.trim()
					? `- The MyPaas public service is \`${mainService.trim()}\`. Keep that service name and make it the HTTP entrypoint. It must have a \`build\` section or a valid \`image\` reference.`
					: '- Choose one clear HTTP service as the public service and report its service name for the MyPaas Main service field. This service must have a `build` section or a valid `image` reference.',
				portRequirement,
				'- The public service must listen on `0.0.0.0` (not `127.0.0.1` or `localhost`) inside the container. Prefer `expose` and container ports over fixed host port bindings — MyPaas supplies the host binding and Caddy route automatically.',
				'- Internal services (databases, caches, brokers) must communicate by **Compose service name**, not `localhost`. For example, use `db:3306` instead of `localhost:3306` in connection strings and environment variables.',
				'- Declare all named volumes in the **top-level `volumes:` section**. For example, if a service uses `db_data:/var/lib/mysql`, there must be a `volumes: { db_data: }` block at the root of the compose file.',
				'- Add healthchecks to infrastructure services. Use generous timeouts for databases that need first-time initialization:',
				'  - MySQL/MariaDB: `start_period: 60s`, `interval: 15s`, `timeout: 10s`, `retries: 10`.',
				'  - PostgreSQL: `start_period: 30s`, `interval: 10s`, `timeout: 5s`, `retries: 10`.',
				'  - Redis/Valkey: `start_period: 5s`, `interval: 5s`, `timeout: 3s`, `retries: 5`.',
				'  - The app service should also have a healthcheck (e.g. `curl -f http://localhost:PORT/health || exit 1`).',
				'- Use `depends_on` with `condition: service_healthy` so the app service waits for its dependencies to be ready before starting. Example: `depends_on: { db: { condition: service_healthy } }`.',
				'- Do not use `container_name`, `network_mode: host`, `privileged`, or mount `/var/run/docker.sock`. Avoid `external` networks unless the MyPaas host is explicitly prepared for them.',
				'- Move hardcoded credentials (DB passwords, secrets) from compose.yml to environment variables loaded via `.env`. Create `.env.example` with safe placeholders.'
			];

		return [
			'Prepare this repository for deployment on MyPaas, a self-hosted PaaS.',
			'',
			...(repositoryContext.length > 0 ? ['Repository context:', ...repositoryContext, ''] : []),
			`Deployment mode: ${mode === 'compose' ? 'Docker Compose' : 'Dockerfile'}`,
			'',
			'Work required:',
			'- Inspect the repository, framework, existing scripts, and current deployment files before editing.',
			...modeRequirements,
			'- Keep configuration in environment variables and do not bake credentials, tokens, URLs with secrets, or machine-specific paths into the image.',
			envRequirement,
			'- Preserve current application behavior. Make only the code/config changes required for a reliable production container.',
			'- Update the README with local container commands and the exact MyPaas values to enter: deployment mode, Main service when applicable, and required environment keys.',
			'',
			'Validation:',
			mode === 'compose'
				? '- Run the relevant project checks, build the images, and run `docker compose config` before finishing.'
				: '- Run the relevant project checks and a production `docker build` before finishing.',
			'- Do not deploy, push, or commit unless I explicitly ask. Finish with a concise summary of files changed, validation performed, and the MyPaas settings I should use.'
		].join('\n');
	}

	async function copyHandoffPrompt() {
		if (!handoffPrompt) return;
		try {
			if (!navigator.clipboard) throw new Error('Clipboard API is unavailable');
			await navigator.clipboard.writeText(handoffPrompt);
			copiedHandoffPrompt = handoffPrompt;
			if (handoffCopyTimer) clearTimeout(handoffCopyTimer);
			handoffCopyTimer = setTimeout(() => {
				copiedHandoffPrompt = '';
				handoffCopyTimer = undefined;
			}, 1800);
			toast.success('Coding agent prompt copied');
		} catch {
			toast.error('Failed to copy prompt');
		}
	}

	onDestroy(() => {
		if (handoffCopyTimer) clearTimeout(handoffCopyTimer);
		if (repoInspectTimer) clearTimeout(repoInspectTimer);
	});

	function issueTone(issue: ComposeIssue) {
		if (issue.severity === 'error') return 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-200';
		if (issue.severity === 'warning') return 'border-yellow-200 bg-yellow-50 text-yellow-800 dark:border-yellow-900/60 dark:bg-yellow-950/20 dark:text-yellow-100';
		return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-950/60 dark:text-gray-300';
	}

	function issueLabel(issue: ComposeIssue) {
		return issue.service ? `${issue.severity}: ${issue.service}` : issue.severity;
	}

	async function handleDetectMode(showToast = true): Promise<DeployModeDetection> {
		if (!form.repoUrl.trim()) {
			const message = 'Repository URL is required before detection';
			error = message;
			throw new Error(message);
		}
		if (!repositoryInspectionCurrent) {
			await inspectRepository(false, true);
		}
		if (!form.branch.trim()) {
			const message = 'Select a branch before detection';
			error = message;
			throw new Error(message);
		}

		detecting = true;
		error = '';
		detectMessage = '';
		try {
			const detected = await api.projects.detectMode({
				repoUrl: form.repoUrl,
				branch: form.branch,
				baseDirectory: form.baseDirectory.trim() || undefined
			});
			applyDetectedMode(detected);
			if (showToast) {
				toast.success(`Detected ${detectMessage || detected.deployMode}`);
			}
			return detected;
		} catch (err) {
			detectMessage = '';
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

	async function ensureCurrentRepositoryValidation() {
		if (repositoryInspectionCurrent) return;
		await inspectRepository(false, true);
		if (repoInspectError || lastRepoInspectKey !== repositoryInspectionKey()) {
			throw new Error(repoInspectError || 'Repository validation did not complete for the current source');
		}
	}

	async function handleSubmit() {
		if (submitting || detecting || inspectingRepo) return;
		submitting = true;
		error = '';
		try {
			if (form.sourceType === 'git') {
				await ensureCurrentRepositoryValidation();
			}

			let deployMode = form.sourceType === 'registry' ? 'image' as DeployModeChoice : form.deployMode;
			let mainService = form.mainService || null;
			if (form.sourceType === 'git' && deployMode === 'auto') {
				const detected = await handleDetectMode(false);
				deployMode = detected.deployMode;
				mainService = detected.mainService || mainService;
			}
			if (composeDisabledReason) {
				throw new Error(composeDisabledReason);
			}
			if (deployMode === 'static' || deployMode === 'image') {
				mainService = null;
			}
			if (deployMode === 'static') {
				form.appPort = '80';
				form.sharedPostgres = false;
			}
			const appPort = resolveProjectAppPort(deployMode, form.appPort);

			const envVars = envDrafts
				.filter((item) => normalizeEnvKey(item.key) && item.value.length > 0)
				.map((item) => ({ key: normalizeEnvKey(item.key), value: item.value }));

			const composeFilePath = deployMode === 'compose' ? form.composeFilePath.trim() || null : null;
			const composeOverridePaths = deployMode === 'compose' ? splitCommaList(form.composeOverridePaths) : [];
			const composeProfiles = deployMode === 'compose' ? splitCommaList(form.composeProfiles) : [];
			const composeWorkdir = deployMode === 'compose' ? form.composeWorkdir.trim() || null : null;

			const project = await api.projects.create({
				name: form.name,
				sourceType: form.sourceType,
				repoUrl: form.sourceType === 'git' ? form.repoUrl : '',
				imageRef: form.sourceType === 'registry' ? form.imageRef.trim() : null,
				branch: form.sourceType === 'git' ? form.branch : '',
				deployMode,
				resourceProfile: form.resourceProfile,
				mainService,
				appPort,
				memoryLimitMb: Number(form.memoryMb),
				cpuLimit: Number(form.cpuLimit),
				sharedPostgres: form.sharedPostgres,
				envVars,
				composeFilePath,
				composeOverridePaths,
				composeProfiles,
				composeWorkdir,
				staticFrontendPath: form.sourceType === 'git' ? form.staticFrontendPath || null : null,
				baseDirectory: form.sourceType === 'git' ? form.baseDirectory.trim() || null : null
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

	function splitCommaList(value: string): string[] {
		return value
			.split(',')
			.map((entry) => entry.trim())
			.filter(Boolean);
	}

	function buildPortToServiceMap(services: ComposeServicePlan[]): Map<number, string> {
		const map = new Map<number, string>();
		for (const service of services) {
			for (const port of service.ports ?? []) {
				if (port.target > 0 && !map.has(port.target)) {
					map.set(port.target, service.name);
				}
			}
			for (const port of service.expose ?? []) {
				if (port > 0 && !map.has(port)) {
					map.set(port, service.name);
				}
			}
		}
		return map;
	}

	const LOCALHOST_EXPR = /(?:[a-z]+:\/\/)?(?:localhost|127\.0\.0\.1)(?::(\d+))?/gi;

	function detectLocalhostInEnvDrafts(
		drafts: EnvDraft[],
		portToService: Map<number, string>
	): Map<number, { host: string; port: number; service: string; suggested: string }> {
		const warnings = new Map<number, { host: string; port: number; service: string; suggested: string }>();
		drafts.forEach((draft, index) => {
			const value = draft.value.trim();
			if (!value) return;
			LOCALHOST_EXPR.lastIndex = 0;
			const match = LOCALHOST_EXPR.exec(value);
			if (!match) return;
			const host = match[0];
			const portStr = match[1];
			const port = portStr ? parseInt(portStr, 10) : 0;
			const service = port > 0 ? (portToService.get(port) ?? '') : '';
			const suggested = service
				? value.replace(host.replace(/:\d+$/, ''), service)
				: '';
			warnings.set(index, { host, port, service, suggested });
		});
		return warnings;
	}
</script>

<svelte:head>
	<title>New project · MyPaas</title>
</svelte:head>

<div class="page-shell py-6">
	<Breadcrumbs items={breadcrumbs} />

	<PageHeader
		title="New project"
		description="Create a routable deployment target from source code or a pre-built container image."
	/>

	{#if error}
		<div class="mb-5 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-200">
			<div class="flex items-start justify-between">
				<p class="font-medium">Action blocked</p>
			</div>
			
			{#if error.includes('AI Prompt:\n')}
				{@const parts = error.split('AI Prompt:\n')}
				<p class="mt-1 whitespace-pre-wrap">{parts[0].trim()}</p>
				<div class="mt-4 rounded-md bg-white/60 p-3 shadow-sm ring-1 ring-red-900/10 dark:bg-black/30 dark:ring-white/10 relative group">
					<p class="mb-1 text-[10px] font-bold tracking-wider text-red-800/70 dark:text-red-300/70">SUGGESTED PROMPT</p>
					<p class="font-mono text-xs leading-relaxed text-gray-800 dark:text-gray-200 whitespace-pre-wrap">{parts[1].trim()}</p>
					<div class="absolute right-2 top-2 opacity-0 transition-opacity focus-within:opacity-100 group-hover:opacity-100">
						<button
							type="button"
							class="rounded bg-white p-1.5 text-gray-500 shadow-sm ring-1 ring-gray-900/10 hover:bg-gray-50 hover:text-gray-900 dark:bg-gray-800 dark:text-gray-400 dark:ring-white/10 dark:hover:bg-gray-700 dark:hover:text-white"
							on:click={() => {
								navigator.clipboard.writeText(parts[1].trim());
								toast.success('Prompt copied');
							}}
							title="Copy prompt"
						>
							<Copy size={14} />
						</button>
					</div>
				</div>
			{:else}
				<p class="mt-1 whitespace-pre-wrap">{error}</p>
			{/if}
		</div>
	{/if}

	<div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_24rem]">
		<form class="space-y-5" on:submit|preventDefault={handleSubmit}>
			<SectionPanel
				title="Deployment source"
				description="Choose whether MyPaas should build source code or run an already-built container image."
			>
				<div class="grid gap-4">
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="name">Project name</label>
						<input id="name" type="text" value={form.name} on:input={handleNameInput} placeholder="my-app" class="field w-full" />
						<p class="mt-1 text-[11px] text-gray-500 dark:text-gray-400">Suggested from the source until you edit it. Route preview: <span class="font-mono">{previewHost}</span>.</p>
					</div>

					<SegmentedChoice
						label="Source"
						value={form.sourceType}
						options={sourceTypeOptions}
						on:change={(event) => chooseSourceType(event.detail as SourceType)}
					/>

					{#if form.sourceType === 'git'}
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="repo">Repository URL</label>
							<input
								id="repo"
								type="text"
								value={form.repoUrl}
								placeholder="https://github.com/username/repo"
								class="field w-full font-mono"
								on:input={handleRepoUrlInput}
								on:blur={() => void inspectRepository(false).catch(() => undefined)}
							/>
						</div>
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="branch">Branch</label>
							<div class="flex flex-col gap-2 sm:flex-row">
								<select
									id="branch"
									value={form.branch}
									class="field min-w-0 flex-1 font-mono"
									disabled={inspectingRepo || (!branchOptions.length && !form.branch)}
									on:change={handleBranchChange}
								>
									<option value="" disabled>{inspectingRepo ? 'Loading branches...' : 'Select branch'}</option>
									{#each branchOptions as branch}
										<option value={branch}>{branch}{branch === defaultBranch ? ' (default)' : ''}</option>
									{/each}
								</select>
								<ActionButton
									variant="secondary"
									type="button"
									on:click={() => void inspectRepository(true, true).catch(() => undefined)}
									disabled={inspectingRepo || detecting || !form.repoUrl.trim()}
									loading={inspectingRepo}
									loadingLabel="Loading..."
								>
									Refresh
								</ActionButton>
							</div>
						</div>
						<details class="rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40">
							<summary class="cursor-pointer text-sm font-medium text-gray-950 dark:text-white">Advanced source settings</summary>
							<div class="mt-3">
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="baseDirectory">Project directory</label>
								<input
									id="baseDirectory"
									type="text"
									value={form.baseDirectory}
									placeholder="Leave blank for repository root"
									class="field w-full font-mono"
									on:input={handleBaseDirectoryInput}
									on:blur={() => void inspectRepository(false).catch(() => undefined)}
								/>
								<p class="mt-1 text-[11px] text-gray-500">Only set this for monorepos. Use <code>frontend</code> or <code>apps/api</code>; blank means repository root.</p>
							</div>
						</details>
						{#if repoInspectError}
							<p class="text-xs leading-5 text-red-600 dark:text-red-300">{repoInspectError}</p>
						{/if}
						<div>
							<div class="mb-2 flex items-center justify-between gap-3">
								<p class="text-xs font-medium text-gray-600 dark:text-gray-300">Repository structure</p>
								{#if repoTreeTruncated}<span class="shrink-0 text-[11px] text-gray-500 dark:text-gray-400">First {repoTree.length} entries</span>{/if}
							</div>
							<div class="max-h-72 overflow-auto rounded-md border border-gray-200 bg-white text-xs dark:border-gray-800 dark:bg-gray-950">
								{#if repoTree.length > 0}
									{#each repoTree as item}
										<div class="grid grid-cols-[2.75rem_minmax(0,1fr)] items-center gap-2 border-b border-gray-100 px-3 py-1.5 last:border-b-0 dark:border-gray-900" style={`padding-left: ${0.75 + item.depth * 0.9}rem;`}>
											<span class="rounded border border-gray-200 px-1.5 py-0.5 text-[10px] uppercase text-gray-500 dark:border-gray-800 dark:text-gray-400">{item.type === 'directory' ? 'dir' : 'file'}</span>
											<span class="truncate font-mono {item.type === 'directory' ? 'font-medium text-gray-950 dark:text-white' : 'text-gray-600 dark:text-gray-300'}">{item.path}</span>
										</div>
									{/each}
								{:else}
									<p class="px-3 py-4 text-sm text-gray-500 dark:text-gray-400">{inspectingRepo ? 'Loading repository structure...' : 'Repository structure appears after branches load.'}</p>
								{/if}
							</div>
						</div>
					{:else}
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="imageRef">Container image</label>
							<input id="imageRef" type="text" value={form.imageRef} on:input={handleImageRefInput} placeholder="ghcr.io/example/my-api:v1.4.0" class="field w-full font-mono" autocomplete="off" />
							<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">Public Docker Hub, GHCR, or another OCI-compatible registry. MyPaas pulls this image directly; private registry credentials are not managed in this MVP.</p>
						</div>
						<div class="soft-panel p-3 text-sm">
							<p class="font-medium text-gray-950 dark:text-white">Pre-built image source</p>
							<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">No repository clone or image build runs. Deploy pulls the configured reference, applies env/resource limits, maps the app port, and routes traffic through Caddy.</p>
						</div>
					{/if}
				</div>
			</SectionPanel>

			<SectionPanel
				title="Runtime and entrypoint"
				description="Use detection for repository defaults, then override only the values that need to be explicit."
			>
				<svelte:fragment slot="actions">
					{#if form.sourceType === 'git'}
						<ActionButton variant="secondary" type="button" on:click={() => void handleDetectMode().catch(() => undefined)} disabled={detecting || inspectingRepo || !form.repoUrl.trim() || !form.branch.trim()} loading={detecting} loadingLabel="Detecting...">
							Re-analyze
						</ActionButton>
					{/if}
				</svelte:fragment>
				<div class="space-y-4">
					{#if form.sourceType === 'git'}
						<div class="rounded-md border border-gray-200 bg-gray-50 px-3 py-3 text-sm dark:border-gray-800 dark:bg-gray-950/60" aria-live="polite">
							<div class="flex gap-3">
								<span class="mt-1 h-2.5 w-2.5 shrink-0 rounded-full {detecting || inspectingRepo ? 'bg-yellow-500' : detectMessage ? 'bg-brand-500' : repoInspectError ? 'bg-red-500' : 'bg-gray-400 dark:bg-gray-600'}"></span>
								<div class="min-w-0">
									<p class="font-medium text-gray-950 dark:text-white">{detectionStateLabel}</p>
									<p class="mt-0.5 text-xs leading-5 text-gray-500 dark:text-gray-400">{detectionStateBody}</p>
								</div>
							</div>
						</div>
						<div class="soft-panel p-3 text-sm">
							<p class="font-medium text-gray-950 dark:text-white">Deployment runtime</p>
							<p class="mt-1 text-sm text-gray-950 dark:text-white">{form.deployMode === 'auto' ? 'Analyzing automatically…' : form.deployMode === 'dockerfile' ? 'Dockerfile' : form.deployMode === 'compose' ? 'Docker Compose' : 'Static site'}</p>
							<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">MyPaas uses the repository as the source of truth. Override only when detection is wrong.</p>
						</div>
					{:else}
						<div class="rounded-md border border-brand-500/30 bg-brand-50 px-3 py-3 text-sm dark:border-brand-500/40 dark:bg-brand-500/10">
							<p class="font-medium text-brand-900 dark:text-brand-100">Container image runtime</p>
							<p class="mt-0.5 text-xs leading-5 text-brand-800 dark:text-brand-200">Runtime mode is fixed for registry sources. Repository-based port detection is unavailable for pre-built images in this MVP, so set the container port in Advanced runtime settings. MyPaas still manages host allocation and Caddy routing.</p>
						</div>
					{/if}
				<div class="grid gap-4 sm:grid-cols-2">
					{#if form.deployMode === 'compose'}
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="mainService">Main service</label>
							<input id="mainService" type="text" bind:value={form.mainService} placeholder="app" class="field w-full font-mono" />
						</div>
					{/if}
					{#if form.deployMode !== 'static'}
						<div class="soft-panel p-3 text-sm {form.deployMode === 'compose' ? '' : 'sm:col-span-2'}">
							<p class="font-medium text-gray-950 dark:text-white">Container port</p>
							{#if form.appPort}
								<p class="mt-1 font-mono text-sm text-gray-950 dark:text-white">{form.appPort}</p>
								<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{portStateLabel}. Public traffic still enters through Caddy; this is only the application's internal listening port.</p>
							{:else}
								<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">Not detected yet. MyPaas reads Dockerfile/Compose metadata during runtime detection instead of assuming port 80.</p>
							{/if}
						</div>
					{:else}
						<div class="soft-panel p-3 text-sm sm:col-span-2">
							<p class="font-medium text-gray-950 dark:text-white">Static deployment</p>
							<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">Static projects are served directly by Caddy, so no application port or database runtime is required.</p>
						</div>
					{/if}
				</div>

				{#if form.sourceType === 'git' || form.deployMode !== 'static'}
					<details class="rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40" open={form.sourceType === 'registry' && !form.appPort}>
						<summary class="cursor-pointer text-sm font-medium text-gray-950 dark:text-white">Advanced runtime settings</summary>
						{#if form.sourceType === 'git'}
							<div class="mt-3">
								<SegmentedChoice label="Deployment mode override" value={form.deployMode} options={deployModeOptions} on:change={(event) => chooseDeployMode(event.detail as DeployModeChoice)} />
							</div>
						{/if}
						{#if form.deployMode !== 'static'}
						<div class="mt-3 max-w-sm">
							<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="appPort">Container port override</label>
							<input
								id="appPort"
								type="number"
								min="1"
								max="65535"
								value={form.appPort}
								placeholder="e.g. 3000"
								on:input={handleAppPortInput}
								class="field w-full font-mono"
							/>
							<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">Normally leave this alone. Use it only when repository detection cannot determine the port, or for a pre-built registry image. This is the container port, not Caddy's public port.</p>
						</div>
						{/if}
					</details>
				{/if}
				
				{#if (form.deployMode === 'compose' || form.deployMode === 'dockerfile') && staticFrontendCandidates.length > 0}
					<div class="mt-4 rounded-md border border-brand-200 bg-brand-50 p-4 dark:border-brand-900/60 dark:bg-brand-950/20">
						<div class="mb-3 flex items-start gap-3">
							<div class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-brand-500 text-white">
								<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12l5 5L20 7"/></svg>
							</div>
							<div>
								<p class="text-sm font-medium text-brand-900 dark:text-brand-100">Hybrid Deployment Detected</p>
								<p class="mt-1 text-xs leading-relaxed text-brand-800 dark:text-brand-200">
									MyPaas found a static frontend (e.g. Vite) inside this repository. You can enable Zero-Container Static Deployment to build and serve it using Caddy (0 RAM), alongside your backend API container.
								</p>
							</div>
						</div>
						
						<div>
							<label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-200" for="staticFrontendPath">Deploy Static Frontend</label>
							<select id="staticFrontendPath" bind:value={form.staticFrontendPath} class="field w-full sm:w-1/2">
								<option value="">Disabled (Backend only)</option>
								{#each staticFrontendCandidates as candidate}
									<option value={candidate}>Enabled (Path: {candidate})</option>
								{/each}
							</select>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">If enabled, MyPaas will route <span class="font-mono">/api/*</span> to your backend container, and serve the frontend for all other paths.</p>
						</div>
					</div>
				{/if}

				{#if form.deployMode === 'compose'}
					<details class="rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40">
						<summary class="cursor-pointer text-sm font-medium text-gray-950 dark:text-white">Advanced Compose settings</summary>
						<div class="mt-3">
						<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
							<div>
								<p class="text-xs font-medium text-gray-700 dark:text-gray-200">Compose file</p>
								<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Pick a discovered file or enter a repo-relative path manually. Leave blank to use the top-ranked candidate.</p>
							</div>
							<ActionButton
								variant="secondary"
								size="xs"
								type="button"
								disabled={composeCandidatesLoading || !form.repoUrl.trim() || !form.branch.trim()}
								loading={composeCandidatesLoading}
								loadingLabel="Scanning..."
								on:click={() => void refreshComposeCandidates(true)}
							>
								Scan for compose files
							</ActionButton>
						</div>

						<div class="grid gap-3 sm:grid-cols-2">
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeFilePath">Compose file path</label>
								<input
									id="composeFilePath"
									type="text"
									bind:value={form.composeFilePath}
									placeholder="docker-compose.yml"
									class="field w-full font-mono"
								/>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Repo-relative, forward slashes only (e.g. <span class="font-mono">infra/docker-compose.yml</span>).</p>
							</div>
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeWorkdir">Working directory override</label>
								<input
									id="composeWorkdir"
									type="text"
									bind:value={form.composeWorkdir}
									placeholder="auto (parent of compose file)"
									class="field w-full font-mono"
								/>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Set only if build contexts or env files resolve against a different directory.</p>
							</div>
						</div>

						<div class="mt-3 grid gap-3 sm:grid-cols-2">
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeOverridePaths">Override files</label>
								<input
									id="composeOverridePaths"
									type="text"
									bind:value={form.composeOverridePaths}
									placeholder="docker-compose.prod.yml, docker-compose.cache.yml"
									class="field w-full font-mono"
								/>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Comma-separated, repo-relative. Applied before MyPaas' generated override.</p>
							</div>
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeProfiles">Profiles</label>
								<input
									id="composeProfiles"
									type="text"
									bind:value={form.composeProfiles}
									placeholder="app, worker"
									class="field w-full font-mono"
								/>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Comma-separated <span class="font-mono">COMPOSE_PROFILES</span> values.</p>
							</div>
						</div>

						{#if composeCandidatesError}
							<p class="mt-3 text-xs text-red-600 dark:text-red-300">{composeCandidatesError}</p>
						{/if}
						{#if composeCandidates.length > 0}
							<div class="mt-3">
								<p class="mb-2 text-xs font-medium text-gray-600 dark:text-gray-300">Discovered compose files</p>
								<div class="grid gap-1.5 sm:grid-cols-2">
									{#each composeCandidates as candidate}
										<button
											type="button"
											class={`flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-left text-xs transition
												${form.composeFilePath === candidate.path
													? 'border-brand-500 bg-brand-50 text-brand-900 dark:border-brand-500/60 dark:bg-brand-500/10 dark:text-brand-100'
													: 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-300'}`}
											on:click={() => selectComposeCandidate(candidate.path)}
										>
											<span class="truncate font-mono">{candidate.path}</span>
											<span class="shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-[10px] text-gray-600 dark:bg-gray-900 dark:text-gray-300">
												{candidate.depth === 0 ? 'root' : `depth ${candidate.depth}`}
											</span>
										</button>
									{/each}
								</div>
							</div>
						{/if}
						</div>
					</details>
				{/if}

					{#if error && handoffPrompt}
						<div class="overflow-hidden rounded-md border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950">
							<div class="flex flex-col gap-3 border-b border-gray-100 px-3 py-3 dark:border-gray-800 sm:flex-row sm:items-center sm:justify-between">
								<div class="min-w-0">
									<p class="text-sm font-medium text-gray-950 dark:text-white">Coding agent handoff</p>
									<p class="mt-0.5 text-xs leading-5 text-gray-500 dark:text-gray-400">
										Copy a deployment brief tailored to the selected runtime and current repository fields.
									</p>
								</div>
								<ActionButton variant="secondary" size="xs" type="button" on:click={copyHandoffPrompt}>
									<span class="inline-flex items-center gap-1.5">
										{#if copiedHandoffPrompt === handoffPrompt}
											<Check size={14} aria-hidden="true" />
											Copied
										{:else}
											<Copy size={14} aria-hidden="true" />
											Copy prompt
										{/if}
									</span>
								</ActionButton>
							</div>
							<pre class="max-h-64 overflow-auto whitespace-pre-wrap break-words px-3 py-3 font-mono text-xs leading-5 text-gray-600 dark:text-gray-300">{handoffPrompt}</pre>
						</div>
					{/if}

					{#if composePlan}
						<div class="rounded-md border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950">
							<div class="border-b border-gray-100 px-3 py-3 dark:border-gray-800">
								<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
									<div>
										<p class="text-sm font-medium text-gray-950 dark:text-white">Compose Doctor</p>
										<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Route target: <span class="font-mono">{composePlan.routeTarget}</span></p>
									</div>
									<span
										class="inline-flex w-fit rounded-full border px-2.5 py-1 text-xs font-medium
											{composeBlockingIssues.length > 0 || missingRequiredEnvKeys.length > 0
												? 'border-yellow-200 bg-yellow-50 text-yellow-800 dark:border-yellow-900/60 dark:bg-yellow-950/20 dark:text-yellow-100'
												: 'border-brand-500/30 bg-brand-50 text-brand-900 dark:border-brand-500/40 dark:bg-brand-500/10 dark:text-brand-100'}"
									>
										{composeBlockingIssues.length > 0 || missingRequiredEnvKeys.length > 0 ? 'Needs attention' : 'Ready'}
									</span>
								</div>
							</div>

							<div class="divide-y divide-gray-100 dark:divide-gray-800">
								<div class="grid gap-2 px-3 py-3 text-xs sm:grid-cols-2">
									<div>
										<p class="font-medium text-gray-600 dark:text-gray-300">Recommended public service</p>
										<p class="mt-1 font-mono text-gray-950 dark:text-white">{composePlan.recommendedMainService}:{composePlan.recommendedAppPort}</p>
									</div>
									<div>
										<p class="font-medium text-gray-600 dark:text-gray-300">Required env</p>
										<p class="mt-1 font-mono text-gray-950 dark:text-white">
											{composePlan.requiredEnvVars.length > 0 ? composePlan.requiredEnvVars.join(', ') : '-'}
										</p>
									</div>
								</div>

								<div class="px-3 py-3">
									<p class="mb-2 text-xs font-medium text-gray-600 dark:text-gray-300">Services</p>
									<div class="grid gap-2 md:grid-cols-2">
										{#each composePlan.services as service}
											<div class="rounded-md border border-gray-200 px-3 py-2 text-xs dark:border-gray-800">
												<div class="flex items-center justify-between gap-2">
													<span class="truncate font-mono font-medium text-gray-950 dark:text-white">{service.name}</span>
													<span class="shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-600 dark:bg-gray-900 dark:text-gray-300">{service.role}</span>
												</div>
												<p class="mt-1 truncate text-gray-500 dark:text-gray-400">
													{service.buildContext ? `build: ${service.buildContext}` : service.image ? `image: ${service.image}` : 'no build/image'}
												</p>
												<p class="mt-1 font-mono text-gray-500 dark:text-gray-400">
													ports: {formatComposeServicePorts(service)}
												</p>
											</div>
										{/each}
									</div>
								</div>

								{#if missingRequiredEnvKeys.length > 0}
									<div class="px-3 py-3">
										<p class="mb-2 text-xs font-medium text-red-700 dark:text-red-200">Missing required env values</p>
										<div class="flex flex-wrap gap-1.5">
											{#each missingRequiredEnvKeys as key}
												<span class="rounded border border-red-200 bg-red-50 px-2 py-1 font-mono text-xs text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-200">{key}</span>
											{/each}
										</div>
									</div>
								{/if}

								<div class="px-3 py-3">
									<p class="mb-2 text-xs font-medium text-gray-600 dark:text-gray-300">Issues</p>
									{#if composePlan.issues.length > 0}
										<div class="space-y-2">
											{#each composePlan.issues as issue}
												<div class={`rounded-md border px-3 py-2 text-xs ${issueTone(issue)}`}>
													<p class="font-medium uppercase tracking-normal">{issueLabel(issue)}</p>
													<p class="mt-1 leading-5">{issue.message}</p>
												</div>
											{/each}
										</div>
									{:else}
										<p class="text-xs text-gray-500 dark:text-gray-400">No Compose compatibility issues detected.</p>
									{/if}
								</div>
							</div>
						</div>
					{/if}
				</div>
			</SectionPanel>

			<SectionPanel
				title="Resources"
				description="MyPaas selects a conservative default from the detected runtime; tune it only when needed."
			>
				<div class="soft-panel mb-3 flex flex-col gap-1 p-3 text-sm sm:flex-row sm:items-center sm:justify-between">
					<div>
						<p class="font-medium text-gray-950 dark:text-white">{selectedProfile?.title ?? form.resourceProfile}</p>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Recommended starting allocation.</p>
					</div>
					<p class="font-mono text-xs text-gray-700 dark:text-gray-300">{form.memoryMb} MB · {form.cpuLimit} CPU</p>
				</div>
				<details class="rounded-md border border-gray-200 bg-gray-50/60 p-3 dark:border-gray-800 dark:bg-gray-950/40">
					<summary class="cursor-pointer text-sm font-medium text-gray-950 dark:text-white">Customize resources</summary>
					<div class="mt-3 grid gap-4 sm:grid-cols-3">
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="profile">Profile</label>
						<select id="profile" bind:value={form.resourceProfile} on:change={() => applyResourceProfile(form.resourceProfile)} class="field w-full">
							{#each resourceProfiles as profile}
								<option value={profile.id}>{profile.title}</option>
							{/each}
						</select>
					</div>
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="memory">Memory</label>
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
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="cpu">CPU</label>
						<select id="cpu" bind:value={form.cpuLimit} on:change={markCustomProfile} class="field w-full">
							<option value="0.1">0.10</option>
							<option value="0.2">0.20</option>
							<option value="0.25">0.25</option>
							<option value="0.35">0.35</option>
							<option value="0.5">0.50</option>
							<option value="1">1.00</option>
							<option value="2">2.00</option>
						</select>
					</div>
					</div>
					<p class="mt-3 text-xs text-gray-500 dark:text-gray-400">Changing memory or CPU directly switches the profile to Custom.</p>
				</details>
			</SectionPanel>

			<SectionPanel
				title="Environment"
				description="Add only the variables this project needs. Keys are normalized before create."
			>
				<svelte:fragment slot="actions">
					<div class="flex flex-wrap items-center gap-2">
						<input
							bind:this={envFileInput}
							type="file"
							accept=".env,text/plain"
							class="hidden"
							on:change={handleEnvFileImport}
						/>
						<ActionButton type="button" variant="secondary" size="xs" on:click={triggerEnvFileImport}>
							<span class="inline-flex items-center gap-1.5">
								<Upload class="h-3.5 w-3.5" aria-hidden="true" />
								Import .env
							</span>
						</ActionButton>

					</div>
				</svelte:fragment>
				<div>
					{#if form.deployMode !== 'static'}
						<div class="soft-panel mb-4 flex flex-col gap-3 p-3 text-sm sm:flex-row sm:items-center sm:justify-between">
							<div>
								<p class="font-medium text-gray-950 dark:text-white">Database</p>
								<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Optional managed PostgreSQL for projects that need it.</p>
							</div>
							<label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
								<input type="checkbox" bind:checked={form.sharedPostgres} class="h-4 w-4 rounded border-gray-300 text-gray-950 focus:ring-gray-950 dark:border-gray-700" />
								Use shared PostgreSQL
							</label>
						</div>
					{/if}
					{#if missingRequiredEnvKeys.length > 0}
						<div class="mb-4 rounded-md border border-amber-200 bg-amber-50 px-3 py-3 text-sm text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-100">
							<p class="font-medium">{missingRequiredEnvKeys.length} required environment value{missingRequiredEnvKeys.length === 1 ? '' : 's'} missing</p>
							<p class="mt-1 font-mono text-xs">{missingRequiredEnvKeys.join(', ')}</p>
						</div>
					{/if}
					<div class="overflow-hidden rounded-md border border-gray-200 dark:border-gray-800">
						<div class="hidden gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2 text-[11px] font-medium text-gray-500 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-400 lg:grid lg:grid-cols-[minmax(8rem,1fr)_minmax(10rem,1.4fr)_6rem_2rem]">
							<span>Key</span>
							<span>Value</span>
							<span>Source</span>
							<span></span>
						</div>
						{#if managedDatabaseUrl}
							<div class="grid gap-2 border-b border-gray-100 px-3 py-3 dark:border-gray-800 lg:grid-cols-[minmax(8rem,1fr)_minmax(10rem,1.4fr)_6rem_2rem] lg:items-center">
								<p class="truncate font-mono text-sm font-medium text-gray-950 dark:text-white">DATABASE_URL</p>
								<input value="Generated on create" disabled class="field w-full opacity-70" />
								<span class="truncate text-xs text-gray-500 dark:text-gray-400"><span class="lg:hidden">Source: </span>managed</span>
								<span></span>
							</div>
						{/if}
						{#each envDrafts as draft, index}
							<div class="grid gap-2 border-b border-gray-100 px-3 py-3 last:border-b-0 dark:border-gray-800 lg:grid-cols-[minmax(8rem,1fr)_minmax(10rem,1.4fr)_6rem_2rem] lg:items-center">
								<div class="min-w-0">
									<input
										value={draft.key}
										on:input={(event) => updateEnvDraftKey(index, (event.currentTarget as HTMLInputElement).value)}
										class="field w-full font-mono uppercase"
									/>
									{#if draft.services && draft.services.length > 0}
										<div class="mt-1 flex flex-wrap gap-1">
											{#each draft.services as svc}
												<span class="rounded bg-brand-50 px-1.5 py-0.5 font-mono text-[10px] text-brand-700 dark:bg-brand-500/10 dark:text-brand-200" title={`Used by ${svc}`}>
													{svc}
												</span>
											{/each}
										</div>
									{/if}
								</div>
								<div class="min-w-0">
									<input
										type={draft.sensitive ? 'password' : 'text'}
										value={draft.value}
										on:input={(event) => updateEnvDraftValue(index, (event.currentTarget as HTMLInputElement).value)}
										placeholder={draft.defaultValue ? `sample: ${draft.defaultValue}` : ''}
										class="field w-full font-mono"
									/>
									{#if draft.conflict}
										<p class="mt-1 text-[11px] text-amber-600 dark:text-amber-300">
											Different defaults across services:
											{#each draft.conflict.values as cv, i}
												{#if i > 0}, {/if}<span class="font-mono">{cv.value || '(empty)'}</span> ({cv.sources.join(', ')})
											{/each}
										</p>
									{/if}
								</div>
								<span class="truncate text-xs text-gray-500 dark:text-gray-400" title={draft.source}><span class="lg:hidden">Source: </span>{draft.source}</span>
								<IconButton
									label={`Remove ${draft.key || 'environment variable'}`}
									variant="ghost"
									type="button"
									on:click={() => removeEnvVar(index)}
								>
									<X class="h-4 w-4" aria-hidden="true" />
								</IconButton>
							</div>
							{#if localhostEnvWarnings.has(index)}
								{@const warning = localhostEnvWarnings.get(index)!}
								<div class="border-b border-gray-100 bg-amber-50/60 px-3 py-2 text-xs text-amber-800 dark:border-gray-800 dark:bg-amber-950/20 dark:text-amber-200">
									<p>
										<span class="font-medium">{draft.key}</span> uses <span class="font-mono">{warning.host}</span>.
										In Docker, localhost means the container itself, not another service.
									</p>
									{#if warning.service}
										<p class="mt-1">
											Compose service <span class="font-mono font-medium">{warning.service}</span> exposes port {warning.port}.
											<button
												type="button"
												class="ml-1 underline hover:text-amber-900 dark:hover:text-amber-100"
												on:click={() => updateEnvDraftValue(index, warning.suggested)}
											>
												Use {warning.suggested}
											</button>
										</p>
									{:else}
										<p class="mt-1">Replace <span class="font-mono">localhost</span> with the compose service name (e.g. <span class="font-mono">db</span>, <span class="font-mono">redis</span>, <span class="font-mono">nats</span>).</p>
									{/if}
								</div>
							{/if}
						{/each}
						{#if envDrafts.length === 0 && !managedDatabaseUrl}
							<p class="px-3 py-4 text-sm text-gray-500 dark:text-gray-400">No project environment variables configured.</p>
						{/if}
					</div>
					<div class="mt-3 flex gap-2">
						<input
							value={newEnvKey}
							placeholder="ENV_KEY"
							class="field min-w-0 flex-1 font-mono uppercase"
							on:input={(event) => (newEnvKey = normalizeEnvKey((event.currentTarget as HTMLInputElement).value))}
							on:keydown={handleNewEnvKeydown}
						/>
						<ActionButton type="button" variant="secondary" on:click={addEnvVar}>Add</ActionButton>
					</div>
				</div>
			</SectionPanel>
		</form>

		<aside class="lg:sticky lg:top-6 lg:self-start">
			<SectionPanel
				title="Review"
				description="Confirm the source and deployment plan before create."
				contentClass="p-0"
			>
				<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800" aria-live="polite">
					<span
						class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium
							{canSubmit
								? 'border-brand-500/30 bg-brand-50 text-brand-900 dark:border-brand-500/40 dark:bg-brand-500/10 dark:text-brand-100'
								: 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300'}"
					>
						{reviewStateLabel}
					</span>
					{#if createDisabledReason}
						<p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">{createDisabledReason}</p>
					{/if}
				</div>
				<dl class="divide-y divide-gray-100 text-sm dark:divide-gray-800">
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Subdomain</dt>
						<dd class="mt-1 truncate font-mono font-medium text-gray-950 dark:text-white">{previewHost}</dd>
					</div>
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Source</dt>
						<dd class="mt-1 text-gray-950 dark:text-white">{form.sourceType === 'registry' ? 'Container Registry' : 'Git Repository'}</dd>
					</div>
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">{form.sourceType === 'registry' ? 'Image' : 'Repository'}</dt>
						<dd class="mt-1 truncate font-mono text-gray-950 dark:text-white">{form.sourceType === 'registry' ? (form.imageRef || '-') : (form.repoUrl || '-')}</dd>
					</div>
					<div class="grid grid-cols-2 divide-x divide-gray-100 dark:divide-gray-800">
						<div class="px-5 py-3">
							<dt class="text-xs text-gray-500 dark:text-gray-400">{form.sourceType === 'registry' ? 'Pull strategy' : 'Branch'}</dt>
							<dd class="mt-1 font-mono text-gray-950 dark:text-white">{form.sourceType === 'registry' ? 'docker pull' : (form.branch || '-')}</dd>
						</div>
						<div class="px-5 py-3">
							<dt class="text-xs text-gray-500 dark:text-gray-400">Runtime</dt>
							<dd class="mt-1 font-mono text-gray-950 dark:text-white">{form.deployMode}</dd>
						</div>
					</div>
					<div class="grid grid-cols-3 divide-x divide-gray-100 dark:divide-gray-800">
						<div class="px-5 py-3">
							<dt class="text-xs text-gray-500 dark:text-gray-400">Container port</dt>
							<dd class="mt-1">
								<span class="font-mono text-gray-950 dark:text-white">{form.deployMode === 'static' ? '-' : effectiveAppPort}</span>
								{#if form.deployMode !== 'static'}
									<span class="mt-0.5 block text-[11px] text-gray-500 dark:text-gray-400">{portStateLabel}</span>
								{/if}
							</dd>
						</div>
						<div class="px-5 py-3">
							<dt class="text-xs text-gray-500 dark:text-gray-400">Memory</dt>
							<dd class="mt-1 font-mono text-gray-950 dark:text-white">{form.memoryMb} MB</dd>
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
				{#if form.deployMode === 'compose' && form.composeFilePath}
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Compose file</dt>
						<dd class="mt-1 truncate font-mono text-gray-950 dark:text-white">{form.composeFilePath}</dd>
					</div>
				{/if}
				{#if form.deployMode === 'compose' && form.composeProfiles}
					<div class="px-5 py-3">
						<dt class="text-xs text-gray-500 dark:text-gray-400">Profiles</dt>
						<dd class="mt-1 truncate font-mono text-gray-950 dark:text-white">{form.composeProfiles}</dd>
					</div>
				{/if}
				</dl>
				<div class="border-t border-gray-100 p-5 dark:border-gray-800">
					<ActionButton
						variant="primary"
						size="md"
						type="button"
						full
						on:click={handleSubmit}
						loading={submitting}
						loadingLabel={form.deployMode === 'auto' ? 'Detecting...' : 'Creating...'}
						disabled={!canSubmit}
					>
						Create project
					</ActionButton>
					<p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
						The current repository, branch, and base directory must validate successfully before creation. The server performs a final deployability preflight before saving the project.
					</p>
				</div>
			</SectionPanel>
		</aside>
	</div>
</div>
