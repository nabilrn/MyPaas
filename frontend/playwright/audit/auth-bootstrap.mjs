import { firefox } from '@playwright/test';
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const frontendRoot = path.resolve(__dirname, '../..');
const authFile = path.join(frontendRoot, 'playwright/.auth/user.json');
const baseURL = process.env.MYPAAS_AUDIT_BASE_URL || 'https://nabilrn.space';
const targetURL = new URL('/projects/new', baseURL).toString();

await fs.mkdir(path.dirname(authFile), { recursive: true });

const browser = await firefox.launch({ headless: false });
const context = await browser.newContext();
const page = await context.newPage();

console.log(`Opening ${targetURL}`);
console.log('Complete GitHub OAuth manually if redirected. This waits until MyPaaS is authenticated.');

await page.goto(targetURL, { waitUntil: 'domcontentloaded' });
await page.waitForFunction(() => {
	const path = window.location.pathname;
	const bodyText = document.body?.innerText || '';
	const onCreateProject = path === '/projects/new' && /Create project|New project|Repository URL/i.test(bodyText);
	const onDashboard = /^\/projects(?:\/new)?$/.test(path) && !/Sign in with GitHub|Continue with GitHub/i.test(bodyText);
	return onCreateProject || onDashboard;
}, undefined, { timeout: 10 * 60 * 1000 });

await context.storageState({ path: authFile });
await browser.close();

console.log(`Saved authenticated Playwright storage state to ${path.relative(frontendRoot, authFile)}`);
