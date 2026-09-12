import { defineConfig, devices } from '@playwright/test';

const externalBaseURL = process.env.MYPAAS_AUDIT_BASE_URL;
const baseURL = externalBaseURL || 'http://127.0.0.1:4173';

export default defineConfig({
	testDir: './playwright/audit',
	timeout: 120_000,
	expect: {
		timeout: 10_000
	},
	fullyParallel: false,
	reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
	outputDir: 'test-results',
	webServer: externalBaseURL
		? undefined
		: {
				command: 'pnpm preview --host 127.0.0.1 --port 4173',
				url: baseURL,
				reuseExistingServer: !process.env.CI,
				timeout: 30_000
			},
	use: {
		baseURL,
		trace: 'on',
		screenshot: 'only-on-failure',
		video: 'retain-on-failure'
	},
	projects: [
		{
			name: 'chromium',
			use: {
				...devices['Desktop Chrome'],
				browserName: 'chromium'
			}
		},
		{
			name: 'firefox',
			use: {
				...devices['Desktop Firefox'],
				browserName: 'firefox'
			}
		}
	]
});
