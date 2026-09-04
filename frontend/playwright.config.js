import { defineConfig, devices } from '@playwright/test';

const baseURL = process.env.MYPAAS_AUDIT_BASE_URL || 'http://127.0.0.1:4173';

export default defineConfig({
	testDir: './playwright/audit',
	timeout: 120_000,
	expect: {
		timeout: 10_000
	},
	fullyParallel: false,
	reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
	outputDir: 'test-results',
	use: {
		baseURL,
		trace: 'on',
		screenshot: 'only-on-failure',
		video: 'retain-on-failure'
	},
	projects: [
		{
			name: 'firefox',
			use: {
				...devices['Desktop Firefox'],
				browserName: 'firefox'
			}
		}
	]
});
