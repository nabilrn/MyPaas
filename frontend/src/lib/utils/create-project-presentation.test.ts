import { describe, expect, it } from 'vitest';

import { prioritizeCreateProjectDiagnostics } from './create-project-presentation';
import type { DeployModeDetection } from '$types';

function detectionWithIssues(severities: Array<'error' | 'warning' | 'info'>): DeployModeDetection {
	return {
		deployMode: 'compose',
		branch: 'main',
		defaultBranch: 'main',
		branches: ['main'],
		tree: [],
		treeTruncated: false,
		mainService: 'app',
		services: ['app'],
		composeFile: 'compose.yml',
		hasDockerfile: false,
		envVars: [],
		appPort: 8080,
		composeCandidates: [],
		composePlan: {
			recommendedMainService: 'app',
			recommendedAppPort: 8080,
			routeTarget: 'app:8080',
			requiredEnvVars: [],
			services: [],
			issues: severities.map((severity, index) => ({
				severity,
				code: `ISSUE_${index}`,
				message: `${severity} ${index}`
			}))
		}
	};
}

describe('prioritizeCreateProjectDiagnostics', () => {
	it('keeps every blocking issue while limiting non-blocking diagnostics', () => {
		const result = prioritizeCreateProjectDiagnostics(
			detectionWithIssues(['warning', 'error', 'info', 'warning', 'error', 'info', 'warning'])
		);
		const issues = result.composePlan?.issues ?? [];

		expect(issues.filter((issue) => issue.severity === 'error')).toHaveLength(2);
		expect(issues).toHaveLength(6);
		expect(issues.at(-1)).toMatchObject({ code: 'ADDITIONAL_DIAGNOSTICS' });
	});

	it('leaves already concise diagnostics unchanged', () => {
		const input = detectionWithIssues(['error', 'warning', 'info']);
		expect(prioritizeCreateProjectDiagnostics(input)).toBe(input);
	});
});
