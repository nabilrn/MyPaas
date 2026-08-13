import { firefox } from '@playwright/test';
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const frontendRoot = path.resolve(__dirname, '../..');
const authFile = path.join(frontendRoot, 'playwright/.auth/user.json');
const artifactsRoot = path.join(frontendRoot, 'artifacts/create-project-audit');
const defaultBaseURL = 'https://nabilrn.space';
const defaultRepositoryURL = 'https://github.com/nabilrn/MyPaas';
const defaultImageRef = 'ghcr.io/fluxcd/flux-cli:v2.4.0';
const defaultRegistryPort = '8080';

const productionViewports = [
	{ name: 'desktop', width: 1366, height: 768 },
	{ name: 'large-desktop', width: 1440, height: 900 },
	{ name: 'mobile', width: 390, height: 844 }
];

const mockViewports = [
	{ name: 'desktop', width: 1366, height: 768 }
];

const geometryTargets = [
	{ name: 'main page container', selector: '.page-shell' },
	{ name: 'Create Project form', selector: 'form' },
	{ name: 'source selector', text: 'Source' },
	{ name: 'repository input', label: 'Repository URL' },
	{ name: 'container image input', label: 'Container image' },
	{ name: 'project name', label: 'Project name' },
	{ name: 'analysis timeline', text: 'Preparing project' },
	{ name: 'deployment type', text: 'Deployment type' },
	{ name: 'container port', label: 'Container port' },
	{ name: 'project directory', text: 'Project directory' },
	{ name: 'manual base directory', label: 'Manual path' },
	{ name: 'environment section', text: 'Environment' },
	{ name: 'Advanced trigger', text: 'Advanced settings' },
	{ name: 'Advanced content', selector: 'details[open]' },
	{ name: 'Create Project CTA', role: 'button', namePattern: 'create project' }
];

export async function runAudit({ mode = 'mock' } = {}) {
	const baseURL = process.env.MYPAAS_AUDIT_BASE_URL || defaultBaseURL;
	const startedAt = new Date().toISOString();
	await fs.rm(artifactsRoot, { recursive: true, force: true });
	await fs.mkdir(artifactsRoot, { recursive: true });

	const browser = await firefox.launch({ headless: true });
	const scenarios = buildScenarios(mode);
	const runSummaries = [];

	try {
		for (const scenario of scenarios) {
			const viewports = mode === 'production' && scenario.primary ? productionViewports : mockViewports;
			for (const viewport of viewports) {
				const runName = `${mode}-firefox-${viewport.name}-${scenario.name}`;
				const runDir = path.join(artifactsRoot, runName);
				await fs.mkdir(path.join(runDir, 'checkpoints'), { recursive: true });
				const summary = await runScenario(browser, {
					mode,
					baseURL,
					runName,
					runDir,
					scenario,
					viewport
				});
				runSummaries.push(summary);
			}
		}
	} finally {
		await browser.close();
	}

	const manifest = {
		createdAt: startedAt,
		mode,
		baseURL,
		productionURL: new URL('/projects/new', baseURL).toString(),
		primaryBrowser: 'firefox',
		authState: mode === 'production' ? path.relative(frontendRoot, authFile) : optionalAuthStatePath(),
		artifactRoot: path.relative(frontendRoot, artifactsRoot),
		runs: runSummaries
	};
	const summary = {
		createdAt: startedAt,
		mode,
		baseURL,
		totalRuns: runSummaries.length,
		checkpoints: runSummaries.reduce((count, run) => count + run.checkpoints.length, 0),
		consoleErrors: runSummaries.reduce((count, run) => count + run.consoleErrors, 0),
		consoleWarnings: runSummaries.reduce((count, run) => count + run.consoleWarnings, 0),
		failedRequests: runSummaries.reduce((count, run) => count + run.failedRequests, 0),
		geometryFindings: runSummaries.reduce((count, run) => count + run.geometryFindings, 0),
		runs: runSummaries.map((run) => ({
			name: run.name,
			scenario: run.scenario,
			viewport: run.viewport,
			checkpoints: run.checkpoints,
			readyCheckpoints: run.readyCheckpoints,
			consoleErrors: run.consoleErrors,
			failedRequests: run.failedRequests,
			geometryFindings: run.geometryFindings
		}))
	};

	await writeJSON(path.join(artifactsRoot, 'manifest.json'), manifest);
	await writeJSON(path.join(artifactsRoot, 'summary.json'), summary);
	console.log(`Create Project ${mode} audit complete: ${path.relative(frontendRoot, artifactsRoot)}`);
	return summary;
}

function buildScenarios(mode) {
	if (mode === 'production') {
		const scenarios = [
			{
				name: 'non-destructive-main',
				primary: true,
				sourceType: 'git',
				repositoryURL: process.env.MYPAAS_AUDIT_REPO_URL || defaultRepositoryURL,
				description: 'Real production Create Project flow through repository inspection and detection. Does not submit.'
			},
			{
				name: 'registry-ghcr-ready',
				primary: true,
				sourceType: 'registry',
				imageRef: process.env.MYPAAS_AUDIT_IMAGE_REF || defaultImageRef,
				appPort: process.env.MYPAAS_AUDIT_REGISTRY_PORT || defaultRegistryPort,
				description: 'Real production Container Registry/GHCR flow through image entry, required port, and readiness. Does not submit.'
			},
			{
				name: 'invalid-repository-error',
				sourceType: 'git',
				repositoryURL: 'https://github.com/nabilrn/definitely-missing-create-project-audit-fixture',
				description: 'Safely reproducible production repository error. Does not submit.'
			}
		];
		if (process.env.MYPAAS_AUDIT_SUBDIR_REPO_URL && process.env.MYPAAS_AUDIT_SUBDIR_PATH) {
			scenarios.splice(2, 0, {
				name: 'subdir-base-directory',
				primary: true,
				sourceType: 'git',
				repositoryURL: process.env.MYPAAS_AUDIT_SUBDIR_REPO_URL,
				baseDirectory: process.env.MYPAAS_AUDIT_SUBDIR_PATH,
				description: 'Real production Git flow with Base Directory selection. Does not submit.'
			});
		}
		return scenarios;
	}

	return [
		{ name: 'static-detection', mock: 'static', repositoryURL: 'https://github.com/example/static-site' },
		{ name: 'dockerfile-missing-port', mock: 'missing-port', repositoryURL: 'https://github.com/example/api-no-port' },
		{ name: 'compose-required-env', mock: 'compose-required-env', repositoryURL: 'https://github.com/example/compose-env' },
		{ name: 'compose-doctor-blocker', mock: 'compose-blocker', repositoryURL: 'https://github.com/example/compose-blocker' },
		{ name: 'nested-base-directory', mock: 'nested', repositoryURL: 'https://github.com/example/monorepo', baseDirectory: 'apps/api' },
		{ name: 'slow-repository-inspection', mock: 'slow', repositoryURL: 'https://github.com/example/slow-repo' },
		{ name: 'backend-500', mock: 'backend-500', repositoryURL: 'https://github.com/example/backend-500' },
		{ name: 'timeout', mock: 'timeout', repositoryURL: 'https://github.com/example/timeout' },
		{ name: 'registry-ghcr-ready', sourceType: 'registry', imageRef: defaultImageRef, appPort: defaultRegistryPort },
		{ name: 'project-creation-failure', mock: 'create-failure', repositoryURL: 'https://github.com/example/create-failure', allowSubmit: true }
	];
}

async function runScenario(browser, options) {
	const { mode, baseURL, runName, runDir, scenario, viewport } = options;
	const consoleEvents = [];
	const networkEvents = [];
	const audit = {
		name: runName,
		mode,
		scenario: scenario.name,
		description: scenario.description || scenario.mock,
		baseURL,
		viewport,
		browser: 'firefox',
		nonDestructive: mode === 'production' || !scenario.allowSubmit,
		checkpoints: []
	};

	const contextOptions = {
		baseURL,
		viewport,
		ignoreHTTPSErrors: true,
		recordHar: { path: path.join(runDir, 'network.har'), content: 'omit' }
	};
	const authState = await existingAuthState();
	if (authState) contextOptions.storageState = authState;
	if (mode === 'production' && !authState) {
		throw new Error('Missing authenticated storage state. Run `pnpm audit:auth` from frontend first.');
	}

	const context = await browser.newContext(contextOptions);
	await context.tracing.start({ screenshots: true, snapshots: true, sources: true });
	const page = await context.newPage();
	page.on('console', (message) => {
		if (['error', 'warning'].includes(message.type())) {
			consoleEvents.push({
				type: message.type(),
				text: redact(message.text()),
				location: message.location()
			});
		}
	});
	page.on('requestfinished', async (request) => {
		const response = await request.response();
		const url = request.url();
		if (isRelevantURL(url)) {
			networkEvents.push({
				type: 'finished',
				method: request.method(),
				url: redact(url),
				status: response?.status() ?? 0,
				timing: request.timing()
			});
		}
	});
	page.on('requestfailed', (request) => {
		const url = request.url();
		if (isRelevantURL(url)) {
			networkEvents.push({
				type: 'failed',
				method: request.method(),
				url: redact(url),
				failure: redact(request.failure()?.errorText || 'request failed')
			});
		}
	});

	if (mode === 'mock') await installMockRoutes(page, scenario);

	try {
		await page.goto('/projects/new', { waitUntil: 'domcontentloaded' });
		await page.waitForLoadState('networkidle', { timeout: 20_000 }).catch(() => undefined);
		await checkpoint(page, audit, runDir, '00-initial', consoleEvents, networkEvents);

		if (scenario.sourceType === 'registry') {
			await runRegistryFlow(page, audit, runDir, scenario, consoleEvents, networkEvents);
		} else {
			await runGitFlow(page, audit, runDir, scenario, consoleEvents, networkEvents);
		}
	} finally {
		await context.tracing.stop({ path: path.join(runDir, 'trace.zip') });
		await context.close();
	}

	await writeJSON(path.join(runDir, 'audit.json'), audit);
	await writeJSON(path.join(runDir, 'console.json'), consoleEvents);
	await writeJSON(path.join(runDir, 'network.json'), networkEvents);
	await writeJSON(path.join(runDir, 'geometry.json'), audit.checkpoints.map((item) => ({
		checkpoint: item.name,
		geometry: item.geometry
	})));

	return {
		name: runName,
		scenario: scenario.name,
		viewport,
		artifactDir: path.relative(frontendRoot, runDir),
		checkpoints: audit.checkpoints.map((item) => item.name),
		readyCheckpoints: audit.checkpoints.filter((item) => item.createButton?.enabled).map((item) => item.name),
		consoleErrors: consoleEvents.filter((item) => item.type === 'error').length,
		consoleWarnings: consoleEvents.filter((item) => item.type === 'warning').length,
		failedRequests: networkEvents.filter((item) => item.type === 'failed' || item.status >= 400).length,
		geometryFindings: audit.checkpoints.reduce((count, item) => count + item.geometry.findings.length, 0)
	};
}

async function runGitFlow(page, audit, runDir, scenario, consoleEvents, networkEvents) {
	await fillRepository(page, scenario.repositoryURL);
	await checkpoint(page, audit, runDir, '01-source-entered', consoleEvents, networkEvents);

	await page.waitForTimeout(scenario.mock === 'slow' ? 250 : 120);
	await checkpoint(page, audit, runDir, '02-analyzing', consoleEvents, networkEvents);

	await waitForAnalysisSettled(page, scenario);
	await checkpoint(page, audit, runDir, '03-runtime-detected', consoleEvents, networkEvents);
	await checkpoint(page, audit, runDir, '04-configuration-required', consoleEvents, networkEvents);

	if (scenario.baseDirectory) {
		await toggleAdvanced(page, true);
		await checkpoint(page, audit, runDir, '05-advanced-open', consoleEvents, networkEvents);
		await chooseBaseDirectory(page, scenario.baseDirectory);
		await checkpoint(page, audit, runDir, '06-base-directory-selected', consoleEvents, networkEvents);
		await waitForAnalysisSettled(page, scenario);
		await checkpoint(page, audit, runDir, '07-subdir-runtime-detected', consoleEvents, networkEvents);
		await toggleAdvanced(page, false);
		await checkpoint(page, audit, runDir, '08-advanced-closed', consoleEvents, networkEvents);
		await checkpoint(page, audit, runDir, '09-readiness', consoleEvents, networkEvents);
		return;
	}

	await toggleAdvanced(page, true);
	await checkpoint(page, audit, runDir, '05-advanced-open', consoleEvents, networkEvents);
	await toggleAdvanced(page, false);
	await checkpoint(page, audit, runDir, '06-advanced-closed', consoleEvents, networkEvents);

	const reanalysisResponse = await clickReanalyzeIfAvailable(page);
	await checkpoint(page, audit, runDir, '07-reanalyze-triggered', consoleEvents, networkEvents);
	await waitForAnalysisSettled(page, scenario, reanalysisResponse);
	await checkpoint(page, audit, runDir, '08-readiness', consoleEvents, networkEvents);

	if (scenario.allowSubmit) {
		await clickCreateIfEnabled(page);
		await checkpoint(page, audit, runDir, '09-submitting-or-error', consoleEvents, networkEvents);
	}
}

async function runRegistryFlow(page, audit, runDir, scenario, consoleEvents, networkEvents) {
	await chooseRegistrySource(page);
	await checkpoint(page, audit, runDir, '01-registry-source-selected', consoleEvents, networkEvents);

	await fillImageRef(page, scenario.imageRef || defaultImageRef);
	await checkpoint(page, audit, runDir, '02-image-entered', consoleEvents, networkEvents);
	await checkpoint(page, audit, runDir, '03-port-required', consoleEvents, networkEvents);

	await toggleAdvanced(page, true);
	await checkpoint(page, audit, runDir, '04-advanced-open', consoleEvents, networkEvents);
	await toggleAdvanced(page, false);
	await checkpoint(page, audit, runDir, '05-advanced-closed', consoleEvents, networkEvents);

	await fillContainerPort(page, scenario.appPort || defaultRegistryPort);
	await checkpoint(page, audit, runDir, '06-port-entered', consoleEvents, networkEvents);
	await checkpoint(page, audit, runDir, '07-readiness', consoleEvents, networkEvents);
}

async function checkpoint(page, audit, runDir, name, consoleEvents, networkEvents) {
	const screenshot = path.join('checkpoints', `${name}.png`);
	await page.screenshot({ path: path.join(runDir, screenshot), fullPage: true });
	const [aria, visibleText, controls, createButton, focus, geometry] = await Promise.all([
		captureARIA(page),
		captureVisibleText(page),
		captureVisibleControls(page),
		captureCreateButton(page),
		captureFocus(page),
		captureGeometry(page)
	]);
	audit.checkpoints.push({
		name,
		url: redact(page.url()),
		screenshot,
		aria,
		visibleText,
		controls,
		createButton,
		focus,
		consoleEventCount: consoleEvents.length,
		networkEventCount: networkEvents.length,
		geometry
	});
}

async function captureARIA(page) {
	try {
		return await page.locator('body').ariaSnapshot({ timeout: 2000 });
	} catch (error) {
		return `ARIA snapshot unavailable: ${error.message}`;
	}
}

async function captureVisibleText(page) {
	return page.evaluate(() => {
		const selectors = ['h1', 'h2', 'h3', 'label', 'summary', 'button', '[aria-live]', 'p'];
		return Array.from(document.querySelectorAll(selectors.join(',')))
			.filter((el) => {
				const style = window.getComputedStyle(el);
				const rect = el.getBoundingClientRect();
				return style.visibility !== 'hidden' && style.display !== 'none' && rect.width > 0 && rect.height > 0;
			})
			.map((el) => el.textContent?.replace(/\s+/g, ' ').trim())
			.filter(Boolean)
			.slice(0, 180);
	});
}

async function captureVisibleControls(page) {
	return page.evaluate(() => {
		return Array.from(document.querySelectorAll('button,input,select,textarea,summary,a[href]'))
			.filter((el) => {
				const style = window.getComputedStyle(el);
				const rect = el.getBoundingClientRect();
				return style.visibility !== 'hidden' && style.display !== 'none' && rect.width > 0 && rect.height > 0;
			})
			.map((el) => {
				const rect = el.getBoundingClientRect();
				return {
					tag: el.tagName.toLowerCase(),
					type: el.getAttribute('type') || '',
					text: el.textContent?.replace(/\s+/g, ' ').trim().slice(0, 120) || '',
					label: el.getAttribute('aria-label') || '',
					name: el.getAttribute('name') || '',
					id: el.id || '',
					disabled: Boolean(el.disabled),
					expanded: el.getAttribute('aria-expanded'),
					box: roundBox(rect)
				};
			});
		function roundBox(rect) {
			return {
				x: Math.round(rect.x),
				y: Math.round(rect.y),
				width: Math.round(rect.width),
				height: Math.round(rect.height)
			};
		}
	});
}

async function captureCreateButton(page) {
	const button = page.getByRole('button', { name: /create project/i }).first();
	try {
		const count = await button.count();
		if (!count) return { present: false, enabled: false };
		return {
			present: true,
			enabled: await button.isEnabled(),
			text: await button.innerText().catch(() => ''),
			box: await button.boundingBox()
		};
	} catch {
		return { present: false, enabled: false };
	}
}

async function captureFocus(page) {
	return page.evaluate(() => {
		const el = document.activeElement;
		if (!el) return null;
		return {
			tag: el.tagName.toLowerCase(),
			id: el.id || '',
			name: el.getAttribute('name') || '',
			label: el.getAttribute('aria-label') || '',
			text: el.textContent?.replace(/\s+/g, ' ').trim().slice(0, 120) || ''
		};
	});
}

async function captureGeometry(page) {
	return page.evaluate((targets) => {
		const viewport = { width: window.innerWidth, height: window.innerHeight };
		const pageBox = document.documentElement.getBoundingClientRect();
		const documentWidth = Math.max(
			document.body.scrollWidth,
			document.documentElement.scrollWidth,
			document.body.offsetWidth,
			document.documentElement.offsetWidth
		);
		const boxes = [];
		for (const target of targets) {
			let el = null;
			if (target.selector) el = document.querySelector(target.selector);
			if (!el && target.label) {
				const label = Array.from(document.querySelectorAll('label')).find((item) =>
					item.textContent?.trim().toLowerCase() === target.label.toLowerCase()
				);
				el = label?.htmlFor ? document.getElementById(label.htmlFor) : label;
			}
			if (!el && target.text) {
				el = Array.from(document.querySelectorAll('h1,h2,h3,label,summary,button,p,span')).find((item) =>
					item.textContent?.replace(/\s+/g, ' ').trim().toLowerCase().includes(target.text.toLowerCase())
				);
			}
			if (!el && target.role === 'button') {
				el = Array.from(document.querySelectorAll('button')).find((item) =>
					(item.textContent || item.getAttribute('aria-label') || '').toLowerCase().includes(target.namePattern)
				);
			}
			if (!el) {
				boxes.push({ name: target.name, missing: true });
				continue;
			}
			const rect = el.getBoundingClientRect();
			const style = window.getComputedStyle(el);
			const visible = isVisible(el);
			boxes.push({
				name: target.name,
				tag: el.tagName.toLowerCase(),
				text: el.textContent?.replace(/\s+/g, ' ').trim().slice(0, 120) || '',
				display: style.display,
				box: roundBox(rect),
				visible,
				outsideViewport: rect.right < 0 || rect.left > viewport.width,
				tinyControl: visible && /button|input|select|textarea|summary/i.test(el.tagName) && (rect.width < 32 || rect.height < 28)
			});
		}

		const controls = Array.from(document.querySelectorAll('button,input,select,textarea,summary'))
			.map((el) => ({ el, rect: el.getBoundingClientRect() }))
			.filter((item) => item.rect.width > 0 && item.rect.height > 0 && isVisible(item.el) && item.rect.bottom >= 0 && item.rect.top <= viewport.height);
		const overlaps = [];
		for (let i = 0; i < controls.length; i += 1) {
			for (let j = i + 1; j < controls.length; j += 1) {
				if (controls[i].el.contains(controls[j].el) || controls[j].el.contains(controls[i].el)) continue;
				const a = controls[i].rect;
				const b = controls[j].rect;
				const horizontal = Math.max(0, Math.min(a.right, b.right) - Math.max(a.left, b.left));
				const vertical = Math.max(0, Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top));
				const overlapArea = horizontal * vertical;
				const smallerArea = Math.min(a.width * a.height, b.width * b.height);
				if (overlapArea > 32 && overlapArea / smallerArea > 0.25) {
					overlaps.push({
						first: describe(controls[i].el),
						second: describe(controls[j].el),
						area: Math.round(overlapArea)
					});
				}
			}
		}

		const leftAlignedTargets = boxes.filter((item) =>
			item.box
			&& item.visible
			&& !item.missing
			&& !/cta|advanced content/i.test(item.name)
			&& item.box.width > 240
		);
		const leftEdges = leftAlignedTargets.map((item) => item.box.x);
		const commonLeft = leftEdges.length ? mode(leftEdges) : 0;
		const findings = [];
		if (documentWidth > viewport.width + 1) findings.push({ type: 'horizontal-overflow', documentWidth, viewportWidth: viewport.width });
		for (const box of boxes) {
			if (box.outsideViewport) findings.push({ type: 'outside-viewport', target: box.name, box: box.box });
			if (box.tinyControl) findings.push({ type: 'tiny-control', target: box.name, box: box.box });
			if (box.box && leftAlignedTargets.includes(box) && Math.abs(box.box.x - commonLeft) > 80) {
				findings.push({ type: 'left-alignment-drift', target: box.name, commonLeft, box: box.box });
			}
		}
		for (const overlap of overlaps.slice(0, 20)) findings.push({ type: 'control-overlap', ...overlap });
		return {
			viewport,
			page: roundBox(pageBox),
			documentWidth,
			targets: boxes,
			findings
		};

		function roundBox(rect) {
			return {
				x: Math.round(rect.x),
				y: Math.round(rect.y),
				width: Math.round(rect.width),
				height: Math.round(rect.height),
				right: Math.round(rect.right),
				bottom: Math.round(rect.bottom)
			};
		}
		function describe(el) {
			return `${el.tagName.toLowerCase()}#${el.id || ''} ${(el.textContent || el.getAttribute('aria-label') || '').trim().slice(0, 40)}`.trim();
		}
		function isVisible(el) {
			const style = window.getComputedStyle(el);
			if (style.visibility === 'hidden' || style.display === 'none' || Number(style.opacity) === 0) return false;
			const details = el.closest('details');
			if (details && !details.open && el !== details.querySelector('summary')) return false;
			return true;
		}
		function mode(values) {
			const counts = new Map();
			for (const value of values) counts.set(value, (counts.get(value) || 0) + 1);
			return Array.from(counts.entries()).sort((a, b) => b[1] - a[1])[0]?.[0] || 0;
		}
	}, geometryTargets);
}

async function fillRepository(page, repositoryURL) {
	const input = page.getByLabel('Repository URL').first();
	await input.waitFor({ state: 'visible', timeout: 20_000 });
	await input.fill(repositoryURL);
	await input.dispatchEvent('input', { inputType: 'insertFromPaste', data: repositoryURL });
	await input.blur();
}

async function chooseRegistrySource(page) {
	const button = page.getByRole('button', { name: /container registry/i }).first();
	await button.waitFor({ state: 'visible', timeout: 20_000 });
	await button.click();
	await page.getByRole('textbox', { name: 'Container image' }).waitFor({ state: 'visible', timeout: 10_000 });
}

async function fillImageRef(page, imageRef) {
	const input = page.getByRole('textbox', { name: 'Container image' }).first();
	await input.waitFor({ state: 'visible', timeout: 20_000 });
	await input.fill(imageRef);
	await input.dispatchEvent('input', { inputType: 'insertFromPaste', data: imageRef });
	await input.blur();
}

async function fillContainerPort(page, port) {
	const input = page.getByRole('spinbutton', { name: 'Container port' }).first();
	await input.waitFor({ state: 'visible', timeout: 20_000 });
	await input.fill(port);
	await page.waitForTimeout(150);
}

async function chooseBaseDirectory(page, baseDirectory) {
	const directoryButton = page.getByRole('button', { name: new RegExp(escapeRegex(baseDirectory), 'i') }).first();
	if (await directoryButton.count()) {
		await directoryButton.click();
	} else {
		const input = page.getByLabel('Manual path').first();
		await input.waitFor({ state: 'visible', timeout: 20_000 });
		await input.fill(baseDirectory);
		await input.dispatchEvent('input', { inputType: 'insertFromPaste', data: baseDirectory });
		await input.blur();
	}
	await page.waitForTimeout(150);
}

async function toggleAdvanced(page, open) {
	const advanced = page.locator('details').filter({ hasText: 'Advanced settings' }).first();
	if (!(await advanced.count())) return;
	const summary = advanced.locator('summary').first();
	const detailsOpen = await advanced.evaluate((details) => details.hasAttribute('open')).catch(() => false);
	if ((open && detailsOpen) || (!open && !detailsOpen)) return;
	await summary.click();
	await page.waitForTimeout(150);
}

function escapeRegex(value) {
	return String(value).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

async function clickReanalyzeIfAvailable(page) {
	const button = page.getByRole('button', { name: /re-analyze|try again/i }).first();
	if ((await button.count()) && await button.isEnabled()) {
		const responsePromise = page.waitForResponse(isDeploymentDetectionResponse, { timeout: 20_000 }).catch(() => undefined);
		await button.click();
		await page.waitForTimeout(100);
		return responsePromise;
	}
	return undefined;
}

async function clickCreateIfEnabled(page) {
	const button = page.getByRole('button', { name: /create project/i }).first();
	if ((await button.count()) && await button.isEnabled()) {
		await button.click();
		await page.waitForTimeout(300);
	}
}

async function waitForAnalysisSettled(page, scenario, responsePromise) {
	if (scenario.mock === 'timeout') {
		await page.waitForTimeout(1100);
		return;
	}
	if (responsePromise) {
		await responsePromise;
	} else {
		await page.waitForResponse(isDeploymentDetectionResponse, { timeout: 20_000 }).catch(() => undefined);
	}
	await page.waitForFunction(() => {
		const text = document.body?.innerText || '';
		const hasSettledState = /Ready to create|Needs configuration|Failed|not found|Compose needs attention|Detection could not resolve|required value/i.test(text);
		const isBusy = /Analyzing deployment|Scanning runtime files|Scanning repository for environment variables|Finishing environment scan/i.test(text);
		return hasSettledState && !isBusy;
	}, undefined, { timeout: 20_000 }).catch(() => undefined);
	await page.waitForTimeout(250);
}

function isDeploymentDetectionResponse(response) {
	if (!response.url().includes('/api/projects/detect-mode') || response.request().method() !== 'POST') return false;
	try {
		return response.request().postDataJSON()?.inspectOnly !== true;
	} catch {
		return true;
	}
}

async function installMockRoutes(page, scenario) {
	let detectCalls = 0;
	await page.route('**/api/auth/me', (route) => fulfillJSON(route, {
		data: { id: 'audit-user', email: 'audit@example.test', name: 'Audit User', avatarUrl: '' }
	}));
	await page.route('**/api/me/quota', (route) => fulfillJSON(route, {
		data: { memoryLimitMb: 6144, memoryUsedMb: 512, memoryRuntimeMb: 512, cpuLimit: 3, cpuUsed: 0.5, cpuRuntime: 0.5, projectLimit: 20, projectCount: 3 }
	}));
	await page.route('**/api/projects/detect-compose', (route) => fulfillJSON(route, {
		data: { branch: 'main', defaultBranch: 'main', branches: ['main'], candidates: [{ path: 'docker-compose.yml', score: 100, depth: 0 }] }
	}));
	await page.route('**/api/projects', async (route) => {
		if (route.request().method() !== 'POST') return route.fallback();
		return fulfillJSON(route, {
			error: { code: 'AUDIT_CREATE_FAILURE', message: 'Mocked project creation failure for audit evidence' }
		}, 500);
	});
	await page.route('**/api/projects/detect-mode', async (route) => {
		const request = route.request();
		const body = request.postDataJSON();
		if (scenario.mock === 'timeout') {
			await new Promise((resolve) => setTimeout(resolve, 1200));
			return route.abort('timedout');
		}
		if (scenario.mock === 'slow') await new Promise((resolve) => setTimeout(resolve, 900));
		if (scenario.mock === 'backend-500' && !body?.inspectOnly) {
			return fulfillJSON(route, {
				error: { code: 'AUDIT_BACKEND_500', message: 'Mocked backend 500 during deployment detection' }
			}, 500);
		}
		detectCalls += 1;
		if (body?.inspectOnly) return fulfillJSON(route, { data: mockInspection(scenario.mock) });
		return fulfillJSON(route, { data: mockDetection(scenario.mock, detectCalls) });
	});
}

function fulfillJSON(route, body, status = 200) {
	return route.fulfill({
		status,
		contentType: 'application/json',
		body: JSON.stringify(body)
	});
}

function mockInspection(kind) {
	const tree = kind === 'nested'
		? [
			{ name: 'apps', path: 'apps', type: 'directory', depth: 0 },
			{ name: 'api', path: 'apps/api', type: 'directory', depth: 1 },
			{ name: 'Dockerfile', path: 'apps/api/Dockerfile', type: 'file', depth: 2 }
		]
		: [
			{ name: 'Dockerfile', path: 'Dockerfile', type: 'file', depth: 0 },
			{ name: 'docker-compose.yml', path: 'docker-compose.yml', type: 'file', depth: 0 }
		];
	return {
		branch: 'main',
		defaultBranch: 'main',
		branches: ['main', 'develop'],
		tree,
		treeTruncated: false
	};
}

function mockDetection(kind, detectCalls) {
	const base = {
		...mockInspection(kind),
		mainService: null,
		services: [],
		composeFile: null,
		hasDockerfile: true,
		envVars: [],
		appPort: 3000,
		composePlan: null,
		composeCandidates: [],
		staticFrontendCandidates: []
	};
	if (kind === 'static') return { ...base, deployMode: 'static', hasDockerfile: false, appPort: 80, staticFrontendCandidates: ['dist'] };
	if (kind === 'missing-port') return { ...base, deployMode: 'dockerfile', appPort: 0 };
	if (kind === 'nested') return { ...base, deployMode: 'dockerfile', appPort: 8080 };
	if (kind === 'compose-blocker') {
		return {
			...base,
			deployMode: 'compose',
			mainService: 'api',
			services: ['api', 'db'],
			composeFile: 'docker-compose.yml',
			composePlan: composePlan([{ severity: 'error', code: 'MISSING_PORT', service: 'api', message: 'Compose Doctor could not find a public HTTP port for api.' }])
		};
	}
	if (kind === 'create-failure') return { ...base, deployMode: 'dockerfile', appPort: 3000 };
	if (kind === 'compose-required-env') {
		return {
			...base,
			deployMode: 'compose',
			mainService: 'api',
			services: ['api', 'db'],
			composeFile: 'docker-compose.yml',
			envVars: [
				{ key: 'DATABASE_URL', source: 'compose', sensitive: true, services: ['api'] },
				{ key: 'PUBLIC_URL', source: '.env.example', sensitive: false, services: ['api'] }
			],
			composePlan: composePlan([], ['DATABASE_URL'])
		};
	}
	if (detectCalls > 1) return { ...base, deployMode: 'dockerfile', appPort: 8080 };
	return { ...base, deployMode: 'dockerfile' };
}

function composePlan(issues = [], requiredEnvVars = []) {
	return {
		recommendedMainService: 'api',
		recommendedAppPort: 3000,
		routeTarget: 'api:3000',
		requiredEnvVars,
		services: [
			{
				name: 'api',
				role: 'public',
				buildContext: '.',
				dockerfile: 'Dockerfile',
				image: null,
				ports: [{ target: 3000, published: null, protocol: 'tcp' }],
				expose: [3000],
				dependsOn: ['db']
			},
			{
				name: 'db',
				role: 'internal',
				buildContext: null,
				dockerfile: null,
				image: 'postgres:16',
				ports: [],
				expose: [5432],
				dependsOn: []
			}
		],
		issues
	};
}

async function existingAuthState() {
	try {
		await fs.access(authFile);
		return authFile;
	} catch {
		return undefined;
	}
}

function optionalAuthStatePath() {
	return path.relative(frontendRoot, authFile);
}

function isRelevantURL(url) {
	return /\/api\/projects|\/api\/auth|\/projects\/new/i.test(url);
}

function redact(value) {
	return String(value)
		.replace(/([?&](?:access_token|refresh_token|id_token|token|code|state|jwt|secret)=)[^&\s]+/gi, '$1[redacted]')
		.replace(/(Bearer\s+)[A-Za-z0-9._~+/=-]+/gi, '$1[redacted]')
		.replace(/(github_pat_|gho_|ghu_|ghs_)[A-Za-z0-9_]+/gi, '[redacted-token]');
}

async function writeJSON(file, value) {
	await fs.mkdir(path.dirname(file), { recursive: true });
	await fs.writeFile(file, `${JSON.stringify(value, null, 2)}\n`, 'utf8');
}
