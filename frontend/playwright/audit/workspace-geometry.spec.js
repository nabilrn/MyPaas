import { expect, test } from '@playwright/test';

const owner = {
	id: 'owner-1',
	email: 'owner@example.test',
	githubId: null,
	githubUsername: 'owner',
	avatarUrl: null,
	role: 'owner',
	createdAt: '2026-01-01T00:00:00Z',
	lastLoginAt: '2026-01-01T00:00:00Z'
};

const project = {
	id: 'project-1',
	userId: owner.id,
	name: 'Geometry Project',
	sourceType: 'registry',
	repoUrl: '',
	imageRef: 'ghcr.io/example/geometry:latest',
	branch: 'main',
	subdomain: 'geometry',
	deployMode: 'dockerfile',
	resourceProfile: 'node-python',
	mainService: null,
	appPort: 3000,
	webhookSecret: 'test-secret',
	allocatedPort: 18080,
	memoryLimitMb: 256,
	cpuLimit: 0.35,
	status: 'running',
	activeDeploymentId: null,
	composeFilePath: null,
	composeOverridePaths: [],
	composeProfiles: [],
	composeWorkdir: null,
	serviceResources: {},
	staticFrontendPath: null,
	baseDirectory: null,
	createdAt: '2026-01-01T00:00:00Z',
	updatedAt: '2026-01-01T00:00:00Z'
};

const hostStats = {
	host_ram_bytes: 8 * 1024 * 1024 * 1024,
	host_cpu_cores: 4,
	allocated_ram_mb: 0,
	allocated_cpu: 0,
	memory: {
		total_bytes: 8 * 1024 * 1024 * 1024,
		available_bytes: 6 * 1024 * 1024 * 1024
	},
	cpu: { total_ticks: 10_000, idle_ticks: 8_000 },
	storage: {
		total_bytes: 100 * 1024 * 1024 * 1024,
		available_bytes: 70 * 1024 * 1024 * 1024
	},
	network: { interface: 'eth0', rx_bytes: 1_000, tx_bytes: 2_000 }
};

const settings = {
	build_timeout_minutes: 15,
	profile_static_memory_mb: 64,
	profile_static_cpu_limit: 0.1,
	profile_go_small_memory_mb: 128,
	profile_go_small_cpu_limit: 0.2,
	profile_node_python_memory_mb: 256,
	profile_node_python_cpu_limit: 0.35,
	profile_compose_main_memory_mb: 256,
	profile_compose_main_cpu_limit: 0.35,
	cloudflare_configured: false,
	build_sha: 'geometry-test'
};

const routes = [
	{ name: 'projects', path: '/projects', ready: '.page-shell' },
	{ name: 'containers', path: '/containers', ready: '.page-shell' },
	{ name: 'administration settings', path: '/admin/settings', ready: '.page-shell' },
	{ name: 'project overview', path: '/projects/project-1', ready: '.project-detail-content' },
	{ name: 'project settings', path: '/projects/project-1/settings', ready: '.settings-workspace-contract' }
];

const viewports = [
	{ name: 'desktop-1440x900', width: 1440, height: 900 },
	{ name: 'desktop-1280x720', width: 1280, height: 720 }
];

test.beforeEach(async ({ page }) => {
	await page.route('**/internal/system-update', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				currentSha: 'geometry-test',
				release: null,
				status: { state: 'idle', phase: 'idle', message: '', updatedAt: null }
			})
		});
	});

	await page.route('**/api/**', async (route) => {
		const url = new URL(route.request().url());
		let data;

		switch (url.pathname) {
			case '/api/auth/me':
				data = owner;
				break;
			case '/api/projects':
				data = [];
				break;
			case '/api/projects/project-1':
				data = project;
				break;
			case '/api/projects/project-1/deployments':
				data = [];
				break;
			case '/api/projects/project-1/routes':
				data = [];
				break;
			case '/api/projects/project-1/stream':
				await route.fulfill({ status: 204 });
				return;
			case '/api/admin/host-stats':
				data = hostStats;
				break;
			case '/api/admin/settings':
				data = settings;
				break;
			case '/api/admin/containers':
				data = { containers: [] };
				break;
			default:
				data = [];
		}

		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ data })
		});
	});
});

for (const viewport of viewports) {
	for (const route of routes) {
		test(`${route.name} owns only its real content height at ${viewport.name}`, async ({ page }) => {
			await page.setViewportSize({ width: viewport.width, height: viewport.height });
			await page.goto(route.path, { waitUntil: 'domcontentloaded' });

			const workspace = page.locator('main.app-workspace');
			await expect(workspace).toBeVisible();
			await expect(page.locator(route.ready).first()).toBeVisible();
			await expect(workspace).toHaveAttribute('aria-busy', 'false');

			const geometry = await page.evaluate(() => {
				const shell = document.querySelector('.app-shell');
				const workspace = document.querySelector('main.app-workspace');
				const pageShell = workspace?.querySelector('.page-shell');
				if (!shell || !workspace || !pageShell) throw new Error('workspace geometry target missing');

				const documentTop = window.scrollY;
				const toDocumentBox = (element) => {
					const rect = element.getBoundingClientRect();
					return {
						top: Math.round(rect.top + documentTop),
						bottom: Math.round(rect.bottom + documentTop),
						height: Math.round(rect.height)
					};
				};
				const shellBox = toDocumentBox(shell);
				const workspaceBox = toDocumentBox(workspace);
				const pageBox = toDocumentBox(pageShell);
				const shellIndex = Array.from(document.body.children).indexOf(shell);
				const flowSiblingsAfterShell = Array.from(document.body.children)
					.slice(shellIndex + 1)
					.map((element) => {
						const style = getComputedStyle(element);
						const rect = element.getBoundingClientRect();
						return {
							tag: element.tagName.toLowerCase(),
							id: element.id,
							position: style.position,
							display: style.display,
							height: Math.round(rect.height)
						};
					})
					.filter((item) => item.display !== 'none' && !['fixed', 'absolute'].includes(item.position) && item.height > 0);

				const projectGrid = workspace.querySelector('.grid.lg\\:grid-cols-\\[12rem_minmax\\(0\\,1fr\\)\\]');
				const projectMain = projectGrid?.querySelector(':scope > main');
				const projectAside = projectGrid?.querySelector(':scope > aside');
				const projectLayout = projectGrid && projectMain && projectAside
					? {
						columns: getComputedStyle(projectGrid).gridTemplateColumns,
						mainTop: toDocumentBox(projectMain).top,
						asideTop: toDocumentBox(projectAside).top
					}
					: null;

				return {
					viewport: { width: innerWidth, height: innerHeight },
					largeLayout: matchMedia('(min-width: 1024px)').matches,
					documentScrollHeight: document.documentElement.scrollHeight,
					bodyScrollHeight: document.body.scrollHeight,
					shell: shellBox,
					workspace: workspaceBox,
					page: pageBox,
					projectLayout,
					flowSiblingsAfterShell
				};
			});

			test.info().annotations.push({ type: 'geometry', description: JSON.stringify(geometry) });

			expect(geometry.largeLayout).toBe(true);
			expect(geometry.viewport).toEqual({ width: viewport.width, height: viewport.height });
			expect(geometry.flowSiblingsAfterShell).toEqual([]);
			expect(Math.abs(geometry.documentScrollHeight - geometry.bodyScrollHeight)).toBeLessThanOrEqual(1);
			expect(Math.abs(geometry.documentScrollHeight - geometry.shell.bottom)).toBeLessThanOrEqual(2);
			expect(geometry.workspace.bottom).toBeLessThanOrEqual(geometry.shell.bottom + 1);
			expect(geometry.page.bottom).toBeLessThanOrEqual(geometry.workspace.bottom + 1);
			if (geometry.projectLayout) {
				expect(geometry.projectLayout.columns).not.toBe('none');
				expect(Math.abs(geometry.projectLayout.mainTop - geometry.projectLayout.asideTop)).toBeLessThanOrEqual(1);
			}
		});
	}
}

test('login wordmark keeps its intended emphasis', async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto('/login', { waitUntil: 'domcontentloaded' });

	await expect(page.getByRole('link', { name: 'Continue with GitHub' })).toBeVisible();
	const logo = page.locator('main img[aria-hidden="true"]').first();
	await expect(logo).toBeVisible();

	const box = await logo.boundingBox();
	expect(box).not.toBeNull();
	expect(box.width).toBeGreaterThanOrEqual(220);
	expect(box.height).toBeGreaterThanOrEqual(72);
});
