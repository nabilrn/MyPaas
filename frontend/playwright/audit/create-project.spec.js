import { expect, test } from '@playwright/test';
import { runAudit } from './create-project-audit.mjs';

test('Create Project UX audit harness', async () => {
	const mode = process.env.MYPAAS_AUDIT_MODE || 'mock';
	if (mode === 'production' && !process.env.MYPAAS_AUDIT_BASE_URL) {
		throw new Error('MYPAAS_AUDIT_BASE_URL is required for production UI audits.');
	}
	if (mode === 'production' && !process.env.MYPAAS_AUDIT_REPO_URL) {
		throw new Error('MYPAAS_AUDIT_REPO_URL is required for production Create Project audits.');
	}
	const summary = await runAudit({ mode });
	if (mode !== 'mock') return;

	const run = (scenario) => {
		const match = summary.runs.find((item) => item.scenario === scenario);
		expect(match, `missing mock audit scenario ${scenario}`).toBeTruthy();
		return match;
	};
	const expectNotReadyAt = (scenario, checkpoint) => {
		expect(run(scenario).readyCheckpoints, `${scenario} must not be ready at ${checkpoint}`).not.toContain(checkpoint);
	};
	const expectReadyAt = (scenario, checkpoint) => {
		expect(run(scenario).readyCheckpoints, `${scenario} must be ready at ${checkpoint}`).toContain(checkpoint);
	};

	// Runtime analysis and stale source configuration must fail closed. The
	// nested scenario changes Base Directory after a valid root analysis, so the
	// checkpoint immediately after that change is the stale-analysis contract.
	expectNotReadyAt('slow-repository-inspection', '02-analyzing');
	expectNotReadyAt('nested-base-directory', '06-base-directory-selected');
	expectReadyAt('nested-base-directory', '09-readiness');

	// Unresolved or blocked runtime contracts must never enable creation.
	for (const scenario of [
		'dockerfile-missing-port',
		'compose-required-env',
		'compose-doctor-blocker',
		'backend-500',
		'timeout'
	]) {
		expect(run(scenario).readyCheckpoints, `${scenario} unexpectedly became creatable`).toEqual([]);
	}

	// Valid static and registry contracts should become ready only after their
	// required runtime configuration is actually resolved.
	expectReadyAt('static-detection', '08-readiness');
	expectNotReadyAt('registry-ghcr-ready', '03-port-required');
	expectReadyAt('registry-ghcr-ready', '06-port-entered');
	expectReadyAt('registry-ghcr-ready', '07-readiness');

	// The submit-failure scenario must prove that the mocked backend failure is
	// reached from a legitimately ready form rather than from a forced click.
	expectReadyAt('project-creation-failure', '08-readiness');
});
