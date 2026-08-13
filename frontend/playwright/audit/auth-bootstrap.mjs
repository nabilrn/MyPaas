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
console.log('Complete GitHub OAuth manually if redirected.');
console.log('The browser will stay open until the Create Project form is actually visible.');

await page.goto(targetURL, { waitUntil: 'domcontentloaded' });
const deadline = Date.now() + 10 * 60 * 1000;
while (Date.now() < deadline) {
	if (await page.getByLabel('Repository URL').isVisible().catch(() => false)) break;
	if (new URL(page.url()).pathname === '/projects') {
		await page.goto(targetURL, { waitUntil: 'domcontentloaded' });
		continue;
	}
	await page.waitForTimeout(1000);
}

await page.getByLabel('Repository URL').waitFor({ state: 'visible', timeout: 5000 });

await context.storageState({ path: authFile });
await browser.close();

console.log(`Saved authenticated Playwright storage state to ${path.relative(frontendRoot, authFile)}`);
