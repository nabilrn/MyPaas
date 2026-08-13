import { test } from '@playwright/test';
import { runAudit } from './create-project-audit.mjs';

test('Create Project UX audit harness', async () => {
	await runAudit({ mode: process.env.MYPAAS_AUDIT_MODE || 'mock' });
});
