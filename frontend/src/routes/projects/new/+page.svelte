<script lang="ts">
	import { Check, CircleAlert, Copy, Folder, LoaderCircle, Upload, X } from '@lucide/svelte';
	import { onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import Breadcrumbs from '$components/Breadcrumbs.svelte';
	import IconButton from '$components/IconButton.svelte';
	import InfoDisclosure from '$components/InfoDisclosure.svelte';
	import PageHeader from '$components/PageHeader.svelte';
	import SegmentedChoice from '$components/SegmentedChoice.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import { projectHost, projectURL } from '$lib/utils/urls';
	import {
		projectCreationReadiness,
		projectNameValidationMessage,
		repositoryDirectoryChoices,
		resolveProjectAppPort,
		suggestProjectName
	} from '$lib/validation/project';
	import type {
		ComposeCandidate,
		ComposeIssue,
		ComposePlan,
		ComposePortPlan,
		ComposeServicePlan,
		DeployModeDetection,
		EnvVarDiscovery,
		RepoInspection,
		RepoTreeEntry,
		ResourceProfile
	} from '$types';

	type SourceType = 'git' | 'registry';
	type DeployModeChoice = 'auto' | 'dockerfile' | 'compose' | 'static' | 'image';
	type EnvDraft = EnvVarDiscovery & { value: string };
	type PortSource = 'unresolved' | 'detected' | 'manual' | 'static';
	type AnalysisStepState = 'complete' | 'active' | 'pending' | 'attention' | 'error';
	type AnalysisStep = { label: string; detail: string; state: AnalysisStepState };
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

	let submitting = false;
	let detecting = false;
	let inspectingRepo = false;
	let error = '';
	let detectError = '';
	let detectMessage = '';
	let repoInspectError = '';
	let repoInspectMessage = '';
	let repoInspectTimer: ReturnType<typeof setTimeout> | undefined;
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
	let staticFrontendCandidates: string[] = [];
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

	$: previewHost = projectHost(form.name || 'your-app', $page.url.hostname);
	$: previewOrigin = projectURL(form.name || 'your-app', $page.url.protocol, $page.url.hostname);
	$: managedDatabaseUrl = form.sharedPostgres && form.deployMode !== 'static';
	$: deployModeOptions = deployModes.map((mode) => ({ value: mode.id, label: mode.title, description: mode.body }));
	$: portStateLabel = form.deployMode === 'static'
		? 'Managed by Caddy'
		: appPortSource === 'detected'
			? 'Detected from repository'
			: appPortSource === 'manual'
				? 'Manual override'
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
	$: requiredEnvKeySet = new Set(normalizedComposeRequiredEnvKeys);
	$: missingRequiredEnvKeys = normalizedComposeRequiredEnvKeys
		.filter((key) => !(managedDatabaseUrl && key === 'DATABASE_URL'))
		.filter((key) => !((envDraftValueByKey.get(key)?.trim()?.length ?? 0) > 0));
	$: composeDisabledReason = composeBlockingIssues[0]?.message
		?? (missingRequiredEnvKeys.length > 0
			? `Fill required env values: ${missingRequiredEnvKeys.slice(0, 3).join(', ')}${missingRequiredEnvKeys.length > 3 ? '...' : ''}`
			: '');
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
		mainService: form.mainService,
		appPort: form.appPort,
		composeDisabledReason,
		busy: submitting || detecting || inspectingRepo
	});
	$: canSubmit = creationReadiness.ready;
	$: createDisabledReason = form.sourceType === 'git' && !form.repoUrl.trim()
		? 'Add a repository URL to begin.'
		: form.sourceType === 'git' && repoInspectError
			? repoInspectError
			: creationReadiness.reason;
	$: nameError = projectNameTouched ? projectNameValidationMessage(form.name) : '';
	$: directoryChoices = repositoryDirectoryChoices(repoTree);
	$: orderedEnvRows = envDrafts
		.map((draft, index) => ({ draft, index, required: requiredEnvKeySet.has(normalizeEnvKey(draft.key)) }))
		.sort((a, b) => Number(b.required) - Number(a.required) || a.draft.key.localeCompare(b.draft.key));
	$: sourceHasValue = form.sourceType === 'git' ? Boolean(form.repoUrl.trim()) : Boolean(form.imageRef.trim());
	$: gitAnalysisSettled = form.sourceType === 'git'
		&& repositoryInspectionCurrent
		&& form.deployMode !== 'auto'
		&& !detecting
		&& !inspectingRepo
		&& !repoInspectError
		&& !detectError;
	$: showAnalysisTimeline = form.sourceType === 'git'
		&& Boolean(form.repoUrl.trim())
		&& !gitAnalysisSettled;
	$: showSetupSummary = form.sourceType === 'registry'
		? Boolean(form.imageRef.trim())
		: gitAnalysisSettled;
	$: runtimeLabel = form.sourceType === 'registry'
		? 'Container image'
		: form.deployMode === 'compose'
			? 'Docker Compose'
			: form.deployMode === 'dockerfile'
				? 'Dockerfile'
				: form.deployMode === 'static'
					? 'Static site'
					: 'Analyzing';
	$: runtimeDetail = form.deployMode === 'static'
		? 'served by Caddy'
		: form.deployMode === 'compose'
			? [form.mainService || 'service required', form.appPort ? `:${form.appPort}` : 'port required'].join(' · ')
			: form.appPort
				? `:${form.appPort}`
				: form.sourceType === 'registry' || form.deployMode !== 'auto'
					? 'port required'
					: '';
	$: backendPromptParts = error.includes('AI Prompt:\n') ? error.split('AI Prompt:\n') : [];
	$: submissionErrorMessage = backendPromptParts.length > 1 ? backendPromptParts[0].trim() : error;
	$: handoffEnvKeys = Array.from(new Set([
		...envDrafts.map((item) => normalizeEnvKey(item.key)).filter(Boolean),
		...(managedDatabaseUrl ? ['DATABASE_URL'] : [])
	])).sort();
	$: generatedHandoffPrompt = buildDeploymentHandoffPrompt(
		form.deployMode,
		form.name,
		form.repoUrl,
		form.branch,
		form.mainService,
		form.appPort,
		appPortSource,
		handoffEnvKeys
	);
	$: actionableHandoffPrompt = backendPromptParts.length > 1
		? backendPromptParts[1].trim()
		: (detectError || composeBlockingIssues.length > 0 ? generatedHandoffPrompt : '');
	$: analysisSteps = buildAnalysisSteps();

	function buildAnalysisSteps(): AnalysisStep[] {
		if (form.sourceType !== 'git') return [];
		const nameMessage = projectNameValidationMessage(form.name);
		const repositoryState: AnalysisStepState = repoInspectError
			? 'error'
			: inspectingRepo
				? 'active'
				: repositoryInspectionCurrent
					? 'complete'
					: form.repoUrl.trim()
						? 'pending'
						: 'pending';
		const deploymentState: AnalysisStepState = detectError
			? 'error'
			: detecting
				? 'active'
				: form.deployMode !== 'auto'
					? 'complete'
					: repositoryInspectionCurrent
						? 'pending'
						: 'pending';
		const configurationState: AnalysisStepState = canSubmit
			? 'complete'
			: form.deployMode !== 'auto' && !detecting && !inspectingRepo
				? 'attention'
				: 'pending';

		return [
			{
				label: 'Source received',
				detail: compactSourceLabel(form.repoUrl),
				state: form.repoUrl.trim() ? 'complete' : 'pending'
			},
			{
				label: 'Project name',
				detail: nameMessage ? nameMessage : form.name,
				state: nameMessage ? 'attention' : 'complete'
			},
			{
				label: 'Repository',
				detail: repoInspectError || (repositoryInspectionCurrent ? repoInspectMessage || 'Repository validated' : 'Checking repository and branches'),
				state: repositoryState
			},
			{
				label: 'Branch',
				detail: form.branch || 'Waiting for repository',
				state: form.branch && repositoryInspectionCurrent ? 'complete' : inspectingRepo ? 'pending' : form.branch ? 'complete' : 'pending'
			},
			{
				label: 'Deployment',
				detail: detectError || (form.deployMode !== 'auto' ? `${runtimeLabel}${runtimeDetail ? ` · ${runtimeDetail}` : ''}` : 'Detecting runtime and deployment defaults'),
				state: deploymentState
			},
			{
				label: 'Configuration',
				detail: canSubmit ? 'Ready to create' : form.deployMode !== 'auto' ? creationReadiness.reason || 'Checking required configuration' : 'Waiting for deployment analysis',
				state: configurationState
			}
		];
	}

	function compactSourceLabel(value: string) {
		const trimmed = value.trim().replace(/\/$/, '').replace(/\.git$/i, '');
		if (!trimmed) return 'Waiting for source';
		const parts = trimmed.split('/');
		return parts.length >= 2 ? parts.slice(-2).join('/') : trimmed;
	}

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
		detectError = '';
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
		detectError = '';
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
		if (detected.branch) form.branch = detected.branch;
		defaultBranch = detected.defaultBranch || defaultBranch;
		branchOptions = normalizeBranches(detected.branches, detected.branch || defaultBranch);
		repoTree = detected.tree ?? repoTree;
		repoTreeTruncated = detected.treeTruncated ?? repoTreeTruncated;
		composePlan = normalizeComposePlan(detected.composePlan);
		composeCandidates = Array.isArray(detected.composeCandidates) ? detected.composeCandidates : [];
		staticFrontendCandidates = Array.isArray(detected.staticFrontendCandidates) ? detected.staticFrontendCandidates : [];
		chooseDeployMode(detected.deployMode, false);
		if (detected.mainService) form.mainService = detected.mainService;
		if (staticFrontendCandidates.length > 0 && !form.staticFrontendPath) {
			form.staticFrontendPath = staticFrontendCandidates[0];
		}
		if (detected.composeFile && !form.composeFilePath) form.composeFilePath = detected.composeFile;
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
		detectError = '';
		detectMessage = detected.deployMode === 'compose'
			? `Docker Compose${detected.composeFile ? ` · ${detected.composeFile}` : ''}`
			: detected.deployMode === 'static'
				? 'Static site'
				: 'Dockerfile';
	}

	async function refreshComposeCandidates(showToast = false) {
		if (form.deployMode !== 'compose') return;
		const repoUrl = form.repoUrl.trim();
		const branch = form.branch.trim();
		if (!repoUrl || !branch) return;
		composeCandidatesLoading = true;
		composeCandidatesError = '';
		try {
			const result = await api.projects.detectCompose({ repoUrl, branch, baseDirectory: form.baseDirectory.trim() || undefined });
			composeCandidates = Array.isArray(result.candidates) ? result.candidates : [];
			if (composeCandidates.length > 0 && !form.composeFilePath) form.composeFilePath = composeCandidates[0].path;
			if (showToast) toast.success(`Found ${composeCandidates.length} compose candidate${composeCandidates.length === 1 ? '' : 's'}`);
		} catch (err) {
			composeCandidates = [];
			composeCandidatesError = err instanceof Error ? err.message : 'Failed to scan for compose files';
			if (showToast) toast.error(composeCandidatesError);
		} finally {
			composeCandidatesLoading = false;
		}
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
		if (ports.length > 0) return ports.map((port) => `${port.published ? `${port.published}:` : ''}${port.target}`).join(', ');
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
		for (const branch of branches ?? []) add(branch);
		add(selected);
		return out;
	}

	function clearDetectedSourceState() {
		repoInspectRequest += 1;
		error = '';
		detectError = '';
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
		error = '';
		if (!projectNameTouched) form.name = suggestProjectName(value);
	}

	function handleBaseDirectoryInput(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value;
		if (value === form.baseDirectory) return;
		form.baseDirectory = value;
		clearDetectedSourceState();
		scheduleRepositoryInspection();
	}

	function chooseProjectDirectory(path: string) {
		if (path === form.baseDirectory) return;
		form.baseDirectory = path;
		clearDetectedSourceState();
		void inspectRepository(false, true).catch(() => undefined);
	}

	function resetRepositoryInspection() {
		clearDetectedSourceState();
		branchOptions = [];
		defaultBranch = '';
	}

	function scheduleRepositoryInspection() {
		if (repoInspectTimer) clearTimeout(repoInspectTimer);
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

	async function reanalyzeSource() {
		if (form.sourceType !== 'git' || inspectingRepo || detecting) return;
		detectError = '';
		repoInspectError = '';
		try {
			await inspectRepository(false, true);
		} catch {
			// The timeline renders the repository error in context.
		}
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
		if (!force && requestKey === lastRepoInspectKey && !repoInspectError) return undefined;

		const requestId = ++repoInspectRequest;
		inspectingRepo = true;
		repoInspectError = '';
		detectError = '';
		try {
			const inspection = await api.projects.inspectRepository({
				repoUrl,
				branch: requestedBranch,
				baseDirectory: form.baseDirectory.trim() || undefined
			});
			if (requestId !== repoInspectRequest) return undefined;
			defaultBranch = inspection.defaultBranch || inspection.branch;
			if (!form.branch.trim() && inspection.branch) form.branch = inspection.branch;
			branchOptions = normalizeBranches(inspection.branches, form.branch || inspection.branch || defaultBranch);
			repoTree = inspection.tree ?? [];
			repoTreeTruncated = inspection.treeTruncated ?? false;
			repoInspectMessage = branchOptions.length === 1 ? '1 branch available' : `${branchOptions.length} branches available`;
			lastRepoInspectKey = repositoryInspectionKey();
			if (showToast) toast.success('Repository validated');
			if (!deployModeManual) {
				setTimeout(() => {
					if (detecting || deployModeManual || lastRepoInspectKey !== repositoryInspectionKey()) return;
					void handleDetectMode(false).catch(() => undefined);
				}, 0);
			}
			return inspection;
		} catch (err) {
			if (requestId !== repoInspectRequest) return undefined;
			const message = err instanceof Error ? err.message : 'Failed to inspect repository';
			repoInspectError = message;
			repoInspectMessage = '';
			repoTree = [];
			repoTreeTruncated = false;
			lastRepoInspectKey = '';
			if (showToast) toast.error(message);
			throw err;
		} finally {
			if (requestId === repoInspectRequest) inspectingRepo = false;
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
			if (line.startsWith('export ')) line = line.slice('export '.length).trim();
			const separatorIndex = line.indexOf('=');
			if (separatorIndex <= 0) {
				skipped += 1;
				continue;
			}
			const key = normalizeEnvKey(line.slice(0, separatorIndex));
			if (!key) {
				skipped += 1;
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
			if (!quote && char === '#' && (index === 0 || /\s/.test(value[index - 1]))) return value.slice(0, index).trimEnd();
		}
		return value;
	}

	function unwrapEnvValue(value: string) {
		if (value.length < 2) return value;
		const quote = value[0];
		if ((quote !== '"' && quote !== "'") || value[value.length - 1] !== quote) return value;
		const inner = value.slice(1, -1);
		if (quote === "'") return inner;
		return inner.replace(/\\n/g, '\n').replace(/\\r/g, '\r').replace(/\\t/g, '\t').replace(/\\"/g, '"').replace(/\\\\/g, '\\');
	}

	function mergeEnvFileVars(vars: EnvDraft[]) {
		const incoming = new Map<string, EnvDraft>();
		for (const item of vars) {
			const key = normalizeEnvKey(item.key);
			if (key) incoming.set(key, { ...item, key });
		}
		const nextDrafts = envDrafts.map((item) => {
			const key = normalizeEnvKey(item.key);
			const imported = incoming.get(key);
			if (!imported) return item;
			incoming.delete(key);
			return { ...item, key, value: imported.value, source: imported.source, sensitive: item.sensitive || imported.sensitive };
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

	async function handleDetectMode(showToast = true): Promise<DeployModeDetection> {
		if (!form.repoUrl.trim()) throw new Error('Repository URL is required before detection');
		if (!repositoryInspectionCurrent) await inspectRepository(false, true);
		if (!form.branch.trim()) throw new Error('Select a branch before detection');

		detecting = true;
		detectError = '';
		detectMessage = '';
		try {
			const detected = await api.projects.detectMode({
				repoUrl: form.repoUrl,
				branch: form.branch,
				baseDirectory: form.baseDirectory.trim() || undefined
			});
			applyDetectedMode(detected);
			if (showToast) toast.success('Project analyzed');
			return detected;
		} catch (err) {
			detectMessage = '';
			detectError = err instanceof Error ? err.message : 'Failed to analyze deployment';
			if (showToast) toast.error(detectError);
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
		projectNameTouched = true;
		if (projectNameValidationMessage(form.name)) return;
		submitting = true;
		error = '';
		try {
			if (form.sourceType === 'git') await ensureCurrentRepositoryValidation();

			let deployMode = form.sourceType === 'registry' ? 'image' as DeployModeChoice : form.deployMode;
			let mainService = form.mainService || null;
			if (form.sourceType === 'git' && deployMode === 'auto') {
				const detected = await handleDetectMode(false);
				deployMode = detected.deployMode;
				mainService = detected.mainService || mainService;
			}
			if (composeDisabledReason) throw new Error(composeDisabledReason);
			if (deployMode === 'static' || deployMode === 'image') mainService = null;
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
			toast.error(submissionErrorMessage || 'Failed to create project');
		} finally {
			submitting = false;
		}
	}

	function splitCommaList(value: string) {
		return value.split(',').map((entry) => entry.trim()).filter(Boolean);
	}

	function buildPortToServiceMap(services: ComposeServicePlan[]) {
		const map = new Map<number, string>();
		for (const service of services) {
			for (const port of service.ports ?? []) if (port.target > 0 && !map.has(port.target)) map.set(port.target, service.name);
			for (const port of service.expose ?? []) if (port > 0 && !map.has(port)) map.set(port, service.name);
		}
		return map;
	}

	const LOCALHOST_EXPR = /(?:[a-z]+:\/\/)?(?:localhost|127\.0\.0\.1)(?::(\d+))?/gi;

	function detectLocalhostInEnvDrafts(drafts: EnvDraft[], portToService: Map<number, string>) {
		const warnings = new Map<number, { host: string; port: number; service: string; suggested: string }>();
		drafts.forEach((draft, index) => {
			const value = draft.value.trim();
			if (!value) return;
			LOCALHOST_EXPR.lastIndex = 0;
			const match = LOCALHOST_EXPR.exec(value);
			if (!match) return;
			const host = match[0];
			const port = match[1] ? parseInt(match[1], 10) : 0;
			const service = port > 0 ? (portToService.get(port) ?? '') : '';
			const suggested = service ? value.replace(host.replace(/:\d+$/, ''), service) : '';
			warnings.set(index, { host, port, service, suggested });
		});
		return warnings;
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
		const portContext = portSource === 'detected' ? 'detected by MyPaas' : portSource === 'manual' ? 'set manually' : 'not resolved yet';
		const portRequirement = appPort.trim()
			? `Keep the application listening on 0.0.0.0:${appPort.trim()} (${portContext}) and expose that container port. MyPaas manages the host port and public route.`
			: 'Make the HTTP application listen on one explicit container port, bind to 0.0.0.0, and declare it with Dockerfile EXPOSE or Compose expose/ports.';
		return [
			'Prepare this repository for deployment on MyPaas, a self-hosted PaaS.',
			'',
			projectName.trim() ? `Project: ${projectName.trim()}` : '',
			repoUrl.trim() ? `Repository: ${repoUrl.trim()}` : '',
			branch.trim() ? `Branch: ${branch.trim()}` : '',
			`Deployment mode: ${mode === 'compose' ? 'Docker Compose' : 'Dockerfile'}`,
			mainService.trim() ? `Public service: ${mainService.trim()}` : '',
			'',
			'Inspect the existing repository before editing and make only the changes required for a reliable production container.',
			portRequirement,
			envKeys.length > 0 ? `Preserve these environment keys without committing secret values: ${envKeys.join(', ')}.` : 'Keep runtime configuration in environment variables and never commit secret values.',
			mode === 'compose' ? 'Validate with the relevant project checks and `docker compose config`.' : 'Validate with the relevant project checks and a production `docker build`.',
			'Do not deploy, push, or commit unless explicitly asked. Finish with the exact MyPaas settings to use.'
		].filter(Boolean).join('\n');
	}

	async function copyHandoffPrompt() {
		if (!actionableHandoffPrompt) return;
		try {
			if (!navigator.clipboard) throw new Error('Clipboard API is unavailable');
			await navigator.clipboard.writeText(actionableHandoffPrompt);
			copiedHandoffPrompt = actionableHandoffPrompt;
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
</script>

<svelte:head>
	<title>New project · MyPaas</title>
</svelte:head>

<div class="page-shell py-6">
	<Breadcrumbs items={breadcrumbs} />
	<div class="mx-auto max-w-4xl">
		<PageHeader title="New project" description="Choose a source. MyPaas will detect the deployment setup and ask only for values it cannot infer." />

		<form class="surface min-w-0 overflow-hidden" on:submit|preventDefault={handleSubmit}>
			<section class="p-5 sm:p-6">
				<div class="mb-5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Source</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Start with what you want to deploy.</p>
				</div>

				<div class="grid gap-5">
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
					{:else}
						<div>
							<div class="mb-1 flex items-center gap-1">
								<label class="block text-xs font-medium text-gray-600 dark:text-gray-300" for="imageRef">Container image</label>
								<InfoDisclosure id="registry-image-info" label="About container images">Use a public Docker Hub, GHCR, or OCI-compatible image reference. Private registry credentials are not managed here yet.</InfoDisclosure>
							</div>
							<input id="imageRef" type="text" value={form.imageRef} on:input={handleImageRefInput} placeholder="ghcr.io/example/my-api:v1.4.0" class="field w-full font-mono" autocomplete="off" />
						</div>
					{/if}

					{#if sourceHasValue}
						<div class="grid gap-4 sm:grid-cols-2">
							<div>
								<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="name">Project name</label>
								<input id="name" type="text" value={form.name} on:input={handleNameInput} placeholder="my-app" class="field w-full" aria-invalid={nameError ? 'true' : undefined} />
								{#if nameError}
									<p class="mt-1 text-xs text-red-600 dark:text-red-300">{nameError}</p>
								{:else}
									<p class="mt-1 truncate font-mono text-[11px] text-gray-500 dark:text-gray-400">{previewHost}</p>
								{/if}
							</div>

							{#if form.sourceType === 'git'}
								<div>
									<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="branch">Branch</label>
									<select id="branch" value={form.branch} class="field w-full font-mono" disabled={inspectingRepo || (!branchOptions.length && !form.branch)} on:change={handleBranchChange}>
										<option value="" disabled>{inspectingRepo ? 'Loading branches…' : 'Select branch'}</option>
										{#each branchOptions as branch}
											<option value={branch}>{branch}{branch === defaultBranch ? ' (default)' : ''}</option>
										{/each}
									</select>
									<p class="mt-1 text-[11px] text-gray-500 dark:text-gray-400">Default branch is selected automatically.</p>
								</div>
							{/if}
						</div>
					{/if}
				</div>
			</section>

			{#if showAnalysisTimeline}
				<section class="border-t border-gray-100 p-5 sm:p-6 dark:border-gray-800" aria-live="polite">
					<div class="mb-5 flex flex-wrap items-start justify-between gap-3">
						<div>
							<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Preparing project</h2>
							<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Progress reflects repository inspection and deployment detection in real time.</p>
						</div>
						{#if !inspectingRepo && !detecting && (repoInspectError || detectError)}
							<ActionButton variant="secondary" size="xs" type="button" on:click={reanalyzeSource}>Try again</ActionButton>
						{/if}
					</div>

					<div class="max-w-2xl">
						{#each analysisSteps as step, index}
							<div class="relative flex gap-3 pb-4 last:pb-0">
								{#if index < analysisSteps.length - 1}
									<span class="absolute left-[0.6875rem] top-6 h-[calc(100%-1rem)] w-px bg-gray-200 dark:bg-gray-800" aria-hidden="true"></span>
								{/if}
								<span class={`relative z-10 mt-0.5 inline-flex h-[1.375rem] w-[1.375rem] shrink-0 items-center justify-center rounded-full border ${step.state === 'complete' ? 'border-brand-600 bg-brand-600 text-white dark:border-brand-400 dark:bg-brand-400 dark:text-gray-950' : step.state === 'active' ? 'border-brand-300 bg-brand-50 text-brand-700 dark:border-brand-700 dark:bg-brand-950/30 dark:text-brand-300' : step.state === 'error' ? 'border-red-300 bg-red-50 text-red-600 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300' : step.state === 'attention' ? 'border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-200' : 'border-gray-200 bg-white text-gray-400 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-600'}`}>
									{#if step.state === 'complete'}
										<Check class="h-3.5 w-3.5" aria-hidden="true" />
									{:else if step.state === 'active'}
										<LoaderCircle class="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" aria-hidden="true" />
									{:else if step.state === 'error' || step.state === 'attention'}
										<CircleAlert class="h-3.5 w-3.5" aria-hidden="true" />
									{:else}
										<span class="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true"></span>
									{/if}
								</span>
								<div class="min-w-0 pt-px">
									<p class="text-sm font-medium text-gray-950 dark:text-white">{step.label}</p>
									<p class={`mt-0.5 text-xs ${step.state === 'error' ? 'text-red-600 dark:text-red-300' : step.state === 'attention' ? 'text-amber-700 dark:text-amber-200' : 'text-gray-500 dark:text-gray-400'}`}>{step.detail}</p>
								</div>
							</div>
						{/each}
					</div>

					{#if detectError && directoryChoices.length > 0}
						<div class="mt-5 border-l-2 border-gray-300 pl-4 dark:border-gray-700">
							<p class="text-sm font-medium text-gray-950 dark:text-white">Application may be inside a subdirectory</p>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Choose a likely application folder and MyPaas will analyze that directory instead.</p>
							<div class="mt-3 flex flex-wrap gap-2">
								{#each directoryChoices.slice(0, 6) as directory}
									<ActionButton variant="secondary" size="xs" type="button" on:click={() => chooseProjectDirectory(directory.path)}>{directory.path}</ActionButton>
								{/each}
							</div>
						</div>
					{/if}
				</section>
			{/if}

			{#if showSetupSummary}
				<section class="border-t border-gray-100 p-5 sm:p-6 dark:border-gray-800" aria-live="polite">
					<div class="rounded-md border border-gray-200 dark:border-gray-800">
						<div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-gray-800">
							<div>
								<div class="flex flex-wrap items-center gap-2">
									<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Deployment setup</h2>
									<span class={`rounded-full px-2 py-0.5 text-[11px] font-medium ${canSubmit ? 'bg-brand-50 text-brand-700 dark:bg-brand-500/10 dark:text-brand-200' : 'bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-200'}`}>{canSubmit ? 'Ready' : 'Needs configuration'}</span>
								</div>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">The detected outcome is summarized here; implementation details stay in Advanced.</p>
							</div>
							{#if form.sourceType === 'git'}
								<ActionButton variant="secondary" size="xs" type="button" on:click={reanalyzeSource} disabled={inspectingRepo || detecting} loading={inspectingRepo || detecting} loadingLabel="Analyzing...">Re-analyze</ActionButton>
							{/if}
						</div>

						<div class="grid gap-px bg-gray-100 dark:bg-gray-800 sm:grid-cols-2">
							<div class="bg-white px-4 py-3 dark:bg-gray-950">
								<p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Project</p>
								<p class="mt-1 truncate text-sm font-medium text-gray-950 dark:text-white">{form.name || '-'}</p>
								<p class="mt-0.5 truncate font-mono text-[11px] text-gray-500 dark:text-gray-400">{previewHost}</p>
							</div>
							<div class="bg-white px-4 py-3 dark:bg-gray-950">
								<p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Source</p>
								<p class="mt-1 truncate font-mono text-sm text-gray-950 dark:text-white">{form.sourceType === 'registry' ? form.imageRef : compactSourceLabel(form.repoUrl)}</p>
								{#if form.sourceType === 'git'}<p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">{form.branch}{form.baseDirectory ? ` · ${form.baseDirectory}` : ' · repository root'}</p>{/if}
							</div>
							<div class="bg-white px-4 py-3 dark:bg-gray-950">
								<p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Runtime</p>
								<p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">{runtimeLabel}</p>
								<p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">{runtimeDetail || portStateLabel}</p>
							</div>
							<div class="bg-white px-4 py-3 dark:bg-gray-950">
								<p class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Environment</p>
								<p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">{envDrafts.length + (managedDatabaseUrl ? 1 : 0)} variable{envDrafts.length + (managedDatabaseUrl ? 1 : 0) === 1 ? '' : 's'}</p>
								<p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">{missingRequiredEnvKeys.length > 0 ? `${missingRequiredEnvKeys.length} required value${missingRequiredEnvKeys.length === 1 ? '' : 's'} missing` : 'No required values missing'}</p>
							</div>
						</div>

						{#if form.deployMode === 'compose' && !form.mainService}
							<div class="border-t border-gray-100 px-4 py-4 dark:border-gray-800">
								<label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-200" for="mainService">Public service</label>
								{#if detectedServices.length > 0}
									<select id="mainService" bind:value={form.mainService} class="field max-w-md font-mono">
										<option value="">Select service</option>
										{#each detectedServices as service}<option value={service}>{service}</option>{/each}
									</select>
								{:else}
									<input id="mainService" type="text" bind:value={form.mainService} placeholder="api" class="field max-w-md font-mono" />
								{/if}
								<p class="mt-1 text-xs text-amber-700 dark:text-amber-200">Choose which service should receive public traffic.</p>
							</div>
						{/if}

						{#if form.deployMode !== 'auto' && form.deployMode !== 'static' && !form.appPort}
							<div class="border-t border-gray-100 px-4 py-4 dark:border-gray-800">
								<div class="mb-1 flex items-center gap-1">
									<label class="block text-xs font-medium text-gray-700 dark:text-gray-200" for="appPort">Container port</label>
									<InfoDisclosure id="container-port-info" label="About container ports">The port your app listens on inside the container. MyPaas manages host allocation and the public route.</InfoDisclosure>
								</div>
								<input id="appPort" type="number" min="1" max="65535" value={form.appPort} placeholder="3000" on:input={handleAppPortInput} class="field max-w-xs font-mono" />
								<p class="mt-1 text-xs text-amber-700 dark:text-amber-200">Detection could not resolve this value automatically.</p>
							</div>
						{/if}

						{#if composeBlockingIssues.length > 0}
							<div class="border-t border-red-100 bg-red-50/50 px-4 py-3 text-xs text-red-700 dark:border-red-900/40 dark:bg-red-950/10 dark:text-red-200">
								<p class="font-medium">Compose needs attention</p>
								<p class="mt-0.5">{composeBlockingIssues[0].message}</p>
							</div>
						{/if}
					</div>
				</section>
			{/if}

			<section class="border-t border-gray-100 p-5 sm:p-6 dark:border-gray-800">
				<div class="mb-4 flex flex-wrap items-start justify-between gap-3">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Environment</h2>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Required values are shown first. Optional variables stay out of the way.</p>
					</div>
					<div>
						<input bind:this={envFileInput} type="file" accept=".env,text/plain" class="hidden" on:change={handleEnvFileImport} />
						<ActionButton type="button" variant="secondary" size="xs" on:click={triggerEnvFileImport}>
							<span class="inline-flex items-center gap-1.5"><Upload class="h-3.5 w-3.5" aria-hidden="true" /> Import .env</span>
						</ActionButton>
					</div>
				</div>

				{#if form.deployMode !== 'static'}
					<div class="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 pb-4 dark:border-gray-800">
						<div class="flex items-center gap-1">
							<span class="text-sm text-gray-700 dark:text-gray-300">Shared PostgreSQL</span>
							<InfoDisclosure id="shared-postgres-info" label="About shared PostgreSQL">Creates a managed PostgreSQL database and injects its connection URL as <span class="font-mono">DATABASE_URL</span>.</InfoDisclosure>
						</div>
						<label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
							<input type="checkbox" bind:checked={form.sharedPostgres} class="h-4 w-4 rounded border-gray-300 text-gray-950 focus:ring-gray-950 dark:border-gray-700" />
							Enable
						</label>
					</div>
				{/if}

				{#if missingRequiredEnvKeys.length > 0}
					<div class="mb-4 border-l-2 border-amber-400 pl-3 text-sm text-amber-900 dark:text-amber-100">
						<p class="font-medium">{missingRequiredEnvKeys.length} required value{missingRequiredEnvKeys.length === 1 ? '' : 's'} missing</p>
						<p class="mt-0.5 font-mono text-xs">{missingRequiredEnvKeys.join(', ')}</p>
					</div>
				{/if}

				{#if managedDatabaseUrl || orderedEnvRows.length > 0}
					<div class="divide-y divide-gray-100 border-y border-gray-100 dark:divide-gray-800 dark:border-gray-800">
						{#if managedDatabaseUrl}
							<div class="grid gap-2 py-3 sm:grid-cols-[minmax(9rem,1fr)_minmax(12rem,1.5fr)_auto] sm:items-center">
								<div><p class="font-mono text-sm font-medium text-gray-950 dark:text-white">DATABASE_URL</p><p class="mt-0.5 text-[11px] text-brand-700 dark:text-brand-300">Managed</p></div>
								<input value="Generated on create" disabled class="field w-full opacity-70" />
								<span class="text-xs text-gray-500 dark:text-gray-400">PostgreSQL</span>
							</div>
						{/if}
						{#each orderedEnvRows as row}
							<div class="py-3">
								<div class="grid gap-2 sm:grid-cols-[minmax(9rem,1fr)_minmax(12rem,1.5fr)_auto] sm:items-start">
									<div class="min-w-0">
										<input value={row.draft.key} on:input={(event) => updateEnvDraftKey(row.index, (event.currentTarget as HTMLInputElement).value)} class="field w-full font-mono uppercase" />
										<div class="mt-1 flex flex-wrap items-center gap-1.5">
											{#if row.required}<span class="text-[11px] font-medium text-amber-700 dark:text-amber-200">Required</span>{/if}
											{#each row.draft.services ?? [] as service}<span class="font-mono text-[10px] text-gray-500 dark:text-gray-400">{service}</span>{/each}
										</div>
									</div>
									<div class="min-w-0">
										<input type={row.draft.sensitive ? 'password' : 'text'} value={row.draft.value} on:input={(event) => updateEnvDraftValue(row.index, (event.currentTarget as HTMLInputElement).value)} placeholder={row.draft.defaultValue ? `sample: ${row.draft.defaultValue}` : ''} class="field w-full font-mono" />
										{#if row.draft.conflict}
											<p class="mt-1 text-[11px] text-amber-600 dark:text-amber-300">Different defaults were detected across services.</p>
										{/if}
									</div>
									<div class="flex items-center justify-between gap-2 sm:justify-end">
										<span class="max-w-28 truncate text-xs text-gray-500 dark:text-gray-400" title={row.draft.source}>{row.draft.source}</span>
										<IconButton label={`Remove ${row.draft.key || 'environment variable'}`} variant="ghost" type="button" on:click={() => removeEnvVar(row.index)}><X class="h-4 w-4" aria-hidden="true" /></IconButton>
									</div>
								</div>
								{#if localhostEnvWarnings.has(row.index)}
									{@const warning = localhostEnvWarnings.get(row.index)!}
									<div class="mt-2 text-xs text-amber-800 dark:text-amber-200">
										<span class="font-medium">{row.draft.key}</span> uses <span class="font-mono">{warning.host}</span>. In Docker, localhost means the current container.
										{#if warning.service}
											<button type="button" class="ml-1 underline" on:click={() => updateEnvDraftValue(row.index, warning.suggested)}>Use {warning.suggested}</button>
										{/if}
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-sm text-gray-500 dark:text-gray-400">No environment variables detected. Add one only if your application needs it.</p>
				{/if}

				<div class="mt-4 flex gap-2">
					<input value={newEnvKey} placeholder="ENV_KEY" aria-label="New environment variable key" class="field min-w-0 flex-1 font-mono uppercase" on:input={(event) => (newEnvKey = normalizeEnvKey((event.currentTarget as HTMLInputElement).value))} on:keydown={handleNewEnvKeydown} />
					<ActionButton type="button" variant="secondary" on:click={addEnvVar}>Add variable</ActionButton>
				</div>
			</section>

			<section class="border-t border-gray-100 dark:border-gray-800">
				<details>
					<summary class="app-focus flex cursor-pointer list-none items-center justify-between px-5 py-4 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-gray-900/50 sm:px-6 [&::-webkit-details-marker]:hidden">
						<span>Advanced</span>
						<span class="text-xs font-normal text-gray-500 dark:text-gray-400">Overrides & diagnostics</span>
					</summary>
					<div class="space-y-7 border-t border-gray-100 px-5 py-5 dark:border-gray-800 sm:px-6">
						{#if form.sourceType === 'git'}
							<div>
								<div class="mb-3 flex items-center gap-1">
									<h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Project directory</h3>
									<InfoDisclosure id="project-directory-info" label="About project directory">Choose the folder containing the application you want to deploy. Repository root works for most projects.</InfoDisclosure>
								</div>
								<div class="overflow-hidden rounded-md border border-gray-200 dark:border-gray-800">
									<button type="button" on:click={() => chooseProjectDirectory('')} class={`flex w-full items-center gap-2 border-b border-gray-100 px-3 py-2 text-left text-sm dark:border-gray-800 ${!form.baseDirectory ? 'bg-brand-50 text-brand-800 dark:bg-brand-500/10 dark:text-brand-200' : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-900'}`}>
										<Folder class="h-4 w-4 shrink-0" aria-hidden="true" />
										<span class="font-medium">Repository root</span>
										{#if !form.baseDirectory}<Check class="ml-auto h-4 w-4" aria-hidden="true" />{/if}
									</button>
									{#if directoryChoices.length > 0}
										<div class="max-h-64 overflow-auto">
											{#each directoryChoices as directory}
												<button type="button" on:click={() => chooseProjectDirectory(directory.path)} class={`flex w-full items-center gap-2 border-b border-gray-100 px-3 py-2 text-left text-sm last:border-b-0 dark:border-gray-800 ${form.baseDirectory === directory.path ? 'bg-brand-50 text-brand-800 dark:bg-brand-500/10 dark:text-brand-200' : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-900'}`} style={`padding-left: ${0.75 + directory.depth * 1.1}rem;`}>
													<Folder class="h-4 w-4 shrink-0" aria-hidden="true" />
													<span class="truncate font-mono text-xs">{directory.name}</span>
													{#if form.baseDirectory === directory.path}<Check class="ml-auto h-4 w-4 shrink-0" aria-hidden="true" />{/if}
												</button>
											{/each}
										</div>
									{:else}
										<p class="px-3 py-3 text-xs text-gray-500 dark:text-gray-400">Folder choices appear after repository validation.</p>
									{/if}
								</div>
								{#if repoTreeTruncated}<p class="mt-1 text-[11px] text-gray-500 dark:text-gray-400">The repository tree was truncated; enter a deeper path manually if needed.</p>{/if}
								<div class="mt-3 max-w-md">
									<label class="mb-1 block text-xs text-gray-500 dark:text-gray-400" for="baseDirectory">Manual path</label>
									<input id="baseDirectory" type="text" value={form.baseDirectory} placeholder="apps/api" class="field w-full font-mono" on:input={handleBaseDirectoryInput} on:blur={() => void inspectRepository(false).catch(() => undefined)} />
								</div>
							</div>
						{/if}

						<div>
							<div class="mb-3 flex items-center gap-1">
								<h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Runtime</h3>
								<InfoDisclosure id="runtime-override-info" label="About runtime overrides">Use these only when automatic detection is wrong or a registry image does not expose enough metadata.</InfoDisclosure>
							</div>
							<div class="grid gap-4">
								{#if form.sourceType === 'git'}
									<SegmentedChoice label="Deployment mode override" value={form.deployMode} options={deployModeOptions} on:change={(event) => chooseDeployMode(event.detail as DeployModeChoice)} />
								{/if}
								<div class="grid gap-4 sm:grid-cols-2">
									{#if form.deployMode === 'compose' && form.mainService}
										<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="mainServiceAdvanced">Public service override</label><input id="mainServiceAdvanced" type="text" bind:value={form.mainService} class="field w-full font-mono" /></div>
									{/if}
									{#if form.deployMode !== 'static' && form.appPort}
										<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="appPortAdvanced">Container port override</label><input id="appPortAdvanced" type="number" min="1" max="65535" value={form.appPort} on:input={handleAppPortInput} class="field w-full font-mono" /></div>
									{/if}
								</div>
								{#if (form.deployMode === 'compose' || form.deployMode === 'dockerfile') && staticFrontendCandidates.length > 0}
									<div class="max-w-md"><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="staticFrontendPath">Static frontend override</label><select id="staticFrontendPath" bind:value={form.staticFrontendPath} class="field w-full"><option value="">Disabled</option>{#each staticFrontendCandidates as candidate}<option value={candidate}>{candidate}</option>{/each}</select></div>
								{/if}
							</div>
						</div>

						{#if form.deployMode === 'compose'}
							<div>
								<div class="mb-3 flex flex-wrap items-center justify-between gap-2">
									<div class="flex items-center gap-1"><h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Compose</h3><InfoDisclosure id="compose-overrides-info" label="About Compose overrides">Override the detected Compose file, working directory, profiles, or additional override files only when repository defaults are not enough.</InfoDisclosure></div>
									<ActionButton variant="secondary" size="xs" type="button" disabled={composeCandidatesLoading || !form.repoUrl.trim() || !form.branch.trim()} loading={composeCandidatesLoading} loadingLabel="Scanning..." on:click={() => void refreshComposeCandidates(true)}>Scan files</ActionButton>
								</div>
								<div class="grid gap-4 sm:grid-cols-2">
									<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeFilePath">Compose file</label><input id="composeFilePath" type="text" bind:value={form.composeFilePath} list="compose-candidates" placeholder="Auto-detect" class="field w-full font-mono" /><datalist id="compose-candidates">{#each composeCandidates as candidate}<option value={candidate.path}></option>{/each}</datalist></div>
									<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeWorkdir">Working directory</label><input id="composeWorkdir" type="text" bind:value={form.composeWorkdir} placeholder="Auto" class="field w-full font-mono" /></div>
									<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeOverridePaths">Override files</label><input id="composeOverridePaths" type="text" bind:value={form.composeOverridePaths} placeholder="docker-compose.prod.yml" class="field w-full font-mono" /></div>
									<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="composeProfiles">Profiles</label><input id="composeProfiles" type="text" bind:value={form.composeProfiles} placeholder="production" class="field w-full font-mono" /></div>
								</div>
								{#if composeCandidatesError}<p class="mt-2 text-xs text-red-600 dark:text-red-300">{composeCandidatesError}</p>{/if}
							</div>
						{/if}

						<div>
							<div class="mb-3 flex items-center gap-1"><h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Resources</h3><InfoDisclosure id="resource-limits-info" label="About resource limits">MyPaas selects a conservative starting profile. Change it only when the workload needs different limits.</InfoDisclosure></div>
							<div class="grid gap-3 sm:grid-cols-3">
								<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="profile">Profile</label><select id="profile" bind:value={form.resourceProfile} on:change={() => applyResourceProfile(form.resourceProfile)} class="field w-full">{#each resourceProfiles as profile}<option value={profile.id}>{profile.title}</option>{/each}</select></div>
								<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="memory">Memory</label><select id="memory" bind:value={form.memoryMb} on:change={markCustomProfile} class="field w-full"><option value="64">64 MB</option><option value="128">128 MB</option><option value="256">256 MB</option><option value="512">512 MB</option><option value="1024">1024 MB</option><option value="2048">2048 MB</option></select></div>
								<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="cpu">CPU</label><select id="cpu" bind:value={form.cpuLimit} on:change={markCustomProfile} class="field w-full"><option value="0.1">0.10</option><option value="0.2">0.20</option><option value="0.25">0.25</option><option value="0.35">0.35</option><option value="0.5">0.50</option><option value="1">1.00</option><option value="2">2.00</option></select></div>
							</div>
						</div>

						{#if form.deployMode === 'compose' && composePlan}
							<details class="border-t border-gray-100 pt-4 dark:border-gray-800">
								<summary class="cursor-pointer text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Compose diagnostics</summary>
								<div class="mt-3 space-y-3">
									<div class="grid gap-2 text-xs sm:grid-cols-2">
										<p><span class="text-gray-500 dark:text-gray-400">Recommended public service</span><br /><span class="font-mono text-gray-950 dark:text-white">{composePlan.recommendedMainService}:{composePlan.recommendedAppPort}</span></p>
										<p><span class="text-gray-500 dark:text-gray-400">Required env</span><br /><span class="font-mono text-gray-950 dark:text-white">{composePlan.requiredEnvVars.length > 0 ? composePlan.requiredEnvVars.join(', ') : '-'}</span></p>
									</div>
									<div class="divide-y divide-gray-100 border-y border-gray-100 dark:divide-gray-800 dark:border-gray-800">
										{#each composePlan.services as service}
											<div class="grid gap-1 py-2 text-xs sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"><div class="min-w-0"><span class="font-mono font-medium text-gray-950 dark:text-white">{service.name}</span><span class="ml-2 text-gray-500 dark:text-gray-400">{service.buildContext ? `build ${service.buildContext}` : service.image ? service.image : 'no build/image'}</span></div><span class="font-mono text-gray-500 dark:text-gray-400">{service.role} · {formatComposeServicePorts(service)}</span></div>
										{/each}
									</div>
									{#each composePlan.issues as issue}
										<div class={`border-l-2 pl-3 text-xs ${issue.severity === 'error' ? 'border-red-500 text-red-700 dark:text-red-200' : issue.severity === 'warning' ? 'border-yellow-500 text-yellow-800 dark:text-yellow-100' : 'border-gray-300 text-gray-600 dark:border-gray-700 dark:text-gray-300'}`}><p class="font-medium">{issue.service ? `${issue.severity}: ${issue.service}` : issue.severity}</p><p class="mt-0.5">{issue.message}</p></div>
									{/each}
								</div>
							</details>
						{/if}

						{#if actionableHandoffPrompt}
							<div class="border-l-2 border-gray-300 pl-3 dark:border-gray-700">
								<div class="flex flex-wrap items-center justify-between gap-2">
									<div><p class="text-xs font-medium text-gray-700 dark:text-gray-200">Coding-agent handoff</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Available only when repository changes may be required.</p></div>
									<ActionButton variant="secondary" size="xs" type="button" on:click={copyHandoffPrompt}><span class="inline-flex items-center gap-1.5"><Copy class="h-3.5 w-3.5" aria-hidden="true" />{copiedHandoffPrompt === actionableHandoffPrompt ? 'Copied' : 'Copy prompt'}</span></ActionButton>
								</div>
							</div>
						{/if}
					</div>
				</details>
			</section>

			<div class="border-t border-gray-100 bg-gray-50/50 px-5 py-4 dark:border-gray-800 dark:bg-gray-900/30 sm:px-6">
				<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0" aria-live="polite">
						<div class="flex items-center gap-2">
							{#if canSubmit}<Check class="h-4 w-4 shrink-0 text-brand-600 dark:text-brand-400" aria-hidden="true" />{:else}<span class="h-2 w-2 shrink-0 rounded-full bg-gray-400 dark:bg-gray-600"></span>{/if}
							<p class="text-sm font-medium text-gray-950 dark:text-white">{creationReadiness.state}</p>
						</div>
						<p class="mt-1 truncate font-mono text-[11px] text-gray-500 dark:text-gray-400">{previewHost}</p>
						{#if submissionErrorMessage}
							<p class="mt-2 text-xs text-red-600 dark:text-red-300">{submissionErrorMessage}</p>
						{:else if createDisabledReason}
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{createDisabledReason}</p>
						{/if}
					</div>
					<ActionButton variant="primary" size="md" type="submit" loading={submitting} loadingLabel="Creating..." disabled={!canSubmit}>Create project</ActionButton>
				</div>
			</div>
		</form>
	</div>
</div>
