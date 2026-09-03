import type { ContainerMetrics, Project } from '$types';

const DEFAULT_COMPOSE_SERVICE_MEMORY_MB = 256;
const DEFAULT_COMPOSE_SERVICE_CPU = 0.25;

export type ProjectResourceScale = {
	memoryMb: number | null;
	cpuPercent: number | null;
};

function mainService(project: Project) {
	return project.mainService?.trim() || 'app';
}

function configuredServiceLimit(project: Project, service: string) {
	if (project.deployMode !== 'compose' || service === mainService(project)) {
		return { memoryMb: project.memoryLimitMb, cpu: project.cpuLimit };
	}

	const override = project.serviceResources?.[service];
	return {
		memoryMb: override?.memoryLimitMb > 0 ? override.memoryLimitMb : DEFAULT_COMPOSE_SERVICE_MEMORY_MB,
		cpu: override?.cpuLimit > 0 ? override.cpuLimit : DEFAULT_COMPOSE_SERVICE_CPU
	};
}

/**
 * Build a shared chart ceiling from the resources assigned to the visible
 * runtime services. Docker stats reports CPU as percent-of-one-core, so a
 * 0.35 CPU allocation maps to a 35% chart ceiling.
 */
export function projectResourceScale(project: Project, metrics: ContainerMetrics[]): ProjectResourceScale {
	if (project.deployMode === 'static' || metrics.length === 0) {
		return { memoryMb: null, cpuPercent: null };
	}

	let memoryMb = 0;
	let cpuPercent = 0;
	for (const metric of metrics) {
		const configured = configuredServiceLimit(project, metric.service);
		const runtimeMemoryLimit = Number.isFinite(metric.memoryLimitMb) && metric.memoryLimitMb > 0
			? metric.memoryLimitMb
			: configured.memoryMb;
		memoryMb = Math.max(memoryMb, runtimeMemoryLimit);
		cpuPercent = Math.max(cpuPercent, configured.cpu * 100);
	}

	return {
		memoryMb: memoryMb > 0 ? memoryMb : null,
		cpuPercent: cpuPercent > 0 ? cpuPercent : null
	};
}

/**
 * Aggregate the allocation represented by the visible runtime services.
 * Usage bars show total current consumption against this total allocation,
 * so Compose services add together rather than sharing one chart ceiling.
 */
export function projectResourceAllocation(project: Project, metrics: ContainerMetrics[]): ProjectResourceScale {
	if (project.deployMode === 'static' || metrics.length === 0) {
		return { memoryMb: null, cpuPercent: null };
	}

	let memoryMb = 0;
	let cpuPercent = 0;
	for (const metric of metrics) {
		const configured = configuredServiceLimit(project, metric.service);
		const runtimeMemoryLimit = Number.isFinite(metric.memoryLimitMb) && metric.memoryLimitMb > 0
			? metric.memoryLimitMb
			: configured.memoryMb;
		memoryMb += runtimeMemoryLimit;
		cpuPercent += configured.cpu * 100;
	}

	return {
		memoryMb: memoryMb > 0 ? memoryMb : null,
		cpuPercent: cpuPercent > 0 ? cpuPercent : null
	};
}
