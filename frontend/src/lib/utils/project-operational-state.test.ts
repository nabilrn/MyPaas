import { describe, expect, it } from 'vitest';

import { deriveProjectOperationalState, type ProjectOperationalStateInput } from './project-operational-state';

type ProjectFixture = ProjectOperationalStateInput['project'];
type DeploymentFixture = NonNullable<ProjectOperationalStateInput['latestDeployment']>;

const baseProject: ProjectFixture = {
	status: 'pending',
	deployMode: 'dockerfile',
	activeDeploymentId: null
};

function project(overrides: Partial<ProjectFixture> = {}): ProjectFixture {
	return { ...baseProject, ...overrides };
}

function deployment(id: string, status: DeploymentFixture['status']): DeploymentFixture {
	return { id, status };
}

describe('project operational state matrix', () => {
	it('derives a never-deployed project as offline and ready to deploy', () => {
		const result = deriveProjectOperationalState({ project: project(), latestDeployment: null });

		expect(result).toMatchObject({
			serving: 'offline',
			release: 'not_deployed',
			desired: 'running',
			headline: 'Not deployed',
			primaryAction: 'deploy',
			attention: 'none'
		});
	});

	it('reports pending plus a failed first deploy as deployment failed and offline', () => {
		const result = deriveProjectOperationalState({
			project: project(),
			latestDeployment: deployment('dep-failed', 'failed')
		});

		expect(result).toMatchObject({
			serving: 'offline',
			release: 'failed',
			desired: 'running',
			headline: 'Deployment failed',
			primaryAction: 'retry',
			attention: 'danger',
			statusLabel: 'Deploy failed'
		});
		expect(result.detail).toContain('no release is serving traffic');
		expect(result.headline).not.toContain('waiting');
	});

	it('keeps the previous release live when a newer deployment fails', () => {
		const result = deriveProjectOperationalState({
			project: project({ status: 'running', activeDeploymentId: 'dep-live' }),
			latestDeployment: deployment('dep-failed', 'failed'),
			runtimeEvidence: 'available'
		});

		expect(result).toMatchObject({
			serving: 'live',
			release: 'failed',
			desired: 'running',
			headline: 'Live; latest deploy failed',
			primaryAction: 'view_deployment',
			attention: 'warning',
			statusLabel: 'Live · deploy failed'
		});
		expect(result.detail).toContain('previous release is still serving traffic');
	});

	it('shows an active first deployment pipeline as deploying and offline until a release exists', () => {
		const result = deriveProjectOperationalState({
			project: project({ status: 'building' }),
			latestDeployment: deployment('dep-building', 'building')
		});

		expect(result).toMatchObject({
			serving: 'offline',
			release: 'deploying',
			desired: 'running',
			headline: 'Deploying',
			primaryAction: 'view_deployment',
			attention: 'info'
		});
	});

	it('keeps an existing release live while a newer deployment is building', () => {
		const result = deriveProjectOperationalState({
			project: project({ status: 'building', activeDeploymentId: 'dep-live' }),
			latestDeployment: deployment('dep-building', 'starting'),
			runtimeEvidence: 'available'
		});

		expect(result).toMatchObject({
			serving: 'live',
			release: 'deploying',
			headline: 'Deploying update',
			primaryAction: 'view_deployment',
			statusLabel: 'Deploying · live'
		});
	});

	it('derives a crashed runtime as offline and directs the operator to logs', () => {
		const result = deriveProjectOperationalState({
			project: project({ status: 'crashed', activeDeploymentId: 'dep-live' }),
			latestDeployment: deployment('dep-live', 'running')
		});

		expect(result).toMatchObject({
			serving: 'offline',
			release: 'succeeded',
			desired: 'running',
			headline: 'Crashed',
			primaryAction: 'view_logs',
			attention: 'danger'
		});
	});

	it('derives an intentionally stopped project as stopped with Start as the next action', () => {
		const result = deriveProjectOperationalState({
			project: project({ status: 'stopped', activeDeploymentId: 'dep-live' }),
			latestDeployment: deployment('dep-live', 'running')
		});

		expect(result).toMatchObject({
			serving: 'offline',
			release: 'succeeded',
			desired: 'stopped',
			headline: 'Stopped',
			primaryAction: 'start',
			attention: 'none'
		});
	});

	it('uses Running for a healthy container-backed project', () => {
		const result = deriveProjectOperationalState({
			project: project({ status: 'running', activeDeploymentId: 'dep-live' }),
			latestDeployment: deployment('dep-live', 'running'),
			runtimeEvidence: 'available'
		});

		expect(result).toMatchObject({
			serving: 'live',
			release: 'succeeded',
			headline: 'Live',
			statusLabel: 'Running',
			statusTone: 'success'
		});
	});

	it('uses Live and Redeploy language for an active static release', () => {
		const result = deriveProjectOperationalState({
			project: project({ status: 'running', deployMode: 'static', activeDeploymentId: 'dep-static' }),
			latestDeployment: deployment('dep-static', 'running'),
			runtimeEvidence: 'not_applicable'
		});

		expect(result).toMatchObject({
			serving: 'live',
			release: 'succeeded',
			desired: 'running',
			headline: 'Live',
			primaryAction: 'deploy',
			primaryActionLabel: 'Redeploy',
			statusLabel: 'Live'
		});
		expect(result.detail).toContain('published');
	});

	it('does not reinterpret unavailable runtime evidence as a crash', () => {
		const result = deriveProjectOperationalState({
			project: project({ status: 'running', activeDeploymentId: 'dep-live' }),
			latestDeployment: deployment('dep-live', 'running'),
			runtimeEvidence: 'unavailable'
		});

		expect(result).toMatchObject({
			serving: 'unknown',
			release: 'succeeded',
			desired: 'running',
			headline: 'Status unknown',
			primaryAction: 'view_logs',
			attention: 'warning',
			statusLabel: 'Unknown'
		});
	});
});
