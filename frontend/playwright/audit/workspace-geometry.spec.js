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
	build_sha: 'geometry-test'
};

const routes = [
	{ name: 'projects', path: '/projects', title: /Projects · MyPaas/i },
	{ name: 'containers', path: '/containers', title: /Containers · MyPaas/i },
	{ name: 'administration settings', path: '/admin/settings', title: /Settings · MyPaaS/i }
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
			await expect(page).toHaveTitle(route.title);
			await expect(page.locator('.page-shell').first()).toBeVisible();
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

				return {
					viewport: { width: innerWidth, height: innerHeight },
					largeLayout: matchMedia('(min-width: 1024px)').matches,
					documentScrollHeight: document.documentElement.scrollHeight,
					bodyScrollHeight: document.body.scrollHeight,
					shell: shellBox,
					workspace: workspaceBox,
					page: pageBox,
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
		});
	}
}
