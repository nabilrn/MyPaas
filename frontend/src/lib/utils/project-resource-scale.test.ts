import { describe, expect, it } from 'vitest';
import type { ContainerMetrics, Project } from '$types';
import { projectResourceAllocation, projectResourceScale } from './project-resource-scale';

function project(overrides: Partial<Project> = {}): Project {
	return {
		id: 'project-1',
		userId: 'user-1',
		name: 'demo',
		sourceType: 'git',
		repoUrl: 'https://github.com/example/demo',
		imageRef: null,
		branch: 'main',
		subdomain: 'demo',
		deployMode: 'dockerfile',
		resourceProfile: 'custom',
		mainService: null,
		appPort: 3000,
		webhookSecret: 'secret',
		allocatedPort: 18080,
		memoryLimitMb: 255,
		cpuLimit: 0.35,
		status: 'running',
		activeDeploymentId: 'deployment-1',
		composeFilePath: null,
		composeOverridePaths: [],
		composeProfiles: [],
		composeWorkdir: null,
		serviceResources: {},
		staticFrontendPath: null,
		baseDirectory: null,
		createdAt: '2026-09-03T00:00:00Z',
		updatedAt: '2026-09-03T00:00:00Z',
		...overrides
	};
}

function metric(overrides: Partial<ContainerMetrics> = {}): ContainerMetrics {
	return {
		service: 'app',
		cpu: 0.08,
		memoryMb: 101,
		memoryLimitMb: 255,
		uptime: '1h',
		...overrides
	};
}

describe('projectResourceScale', () => {
	it('maps project CPU cores to Docker stats percent and uses the runtime memory limit', () => {
		expect(projectResourceScale(project(), [metric()])).toEqual({
			memoryMb: 255,
			cpuPercent: 35
		});
	});

	it('uses the largest visible Compose service allocation for a shared chart scale', () => {
		const compose = project({
			deployMode: 'compose',
			mainService: 'web',
			memoryLimitMb: 256,
			cpuLimit: 0.35,
			serviceResources: { worker: { memoryLimitMb: 768, cpuLimit: 0.8 } }
		});
		const metrics = [
			metric({ service: 'web', memoryLimitMb: 256 }),
			metric({ service: 'worker', memoryLimitMb: 768 })
		];

		expect(projectResourceScale(compose, metrics)).toEqual({ memoryMb: 768, cpuPercent: 80 });
	});

	it('matches backend defaults for Compose secondary services without explicit limits', () => {
		const compose = project({
			deployMode: 'compose',
			mainService: 'web',
			memoryLimitMb: 128,
			cpuLimit: 0.2,
			serviceResources: {}
		});

		expect(projectResourceScale(compose, [metric({ service: 'worker', memoryLimitMb: 0 })])).toEqual({
			memoryMb: 256,
			cpuPercent: 25
		});
	});

	it('does not expose runtime scales for static projects', () => {
		expect(projectResourceScale(project({ deployMode: 'static' }), [metric()])).toEqual({
			memoryMb: null,
			cpuPercent: null
		});
	});
});

describe('projectResourceAllocation', () => {
	it('keeps a single-service allocation equal to the project limit', () => {
		expect(projectResourceAllocation(project(), [metric()])).toEqual({
			memoryMb: 255,
			cpuPercent: 35
		});
	});

	it('adds visible Compose service allocations for aggregate usage bars', () => {
		const compose = project({
			deployMode: 'compose',
			mainService: 'web',
			memoryLimitMb: 256,
			cpuLimit: 0.35,
			serviceResources: { worker: { memoryLimitMb: 768, cpuLimit: 0.8 } }
		});
		const metrics = [
			metric({ service: 'web', memoryLimitMb: 256 }),
			metric({ service: 'worker', memoryLimitMb: 768 })
		];

		expect(projectResourceAllocation(compose, metrics)).toEqual({ memoryMb: 1024, cpuPercent: 115 });
	});

	it('uses backend-compatible defaults when a secondary service has no override', () => {
		const compose = project({
			deployMode: 'compose',
			mainService: 'web',
			memoryLimitMb: 128,
			cpuLimit: 0.2,
			serviceResources: {}
		});
		const metrics = [
			metric({ service: 'web', memoryLimitMb: 128 }),
			metric({ service: 'worker', memoryLimitMb: 0 })
		];

		expect(projectResourceAllocation(compose, metrics)).toEqual({ memoryMb: 384, cpuPercent: 45 });
	});
});
