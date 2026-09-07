import type { ContainerMetrics, Project } from '$types';

const DEFAULT_COMPOSE_SERVICE_MEMORY_MB = 256;

export type ProjectResourceScale = {
	memoryMb: number | null;
	// CPU is host-shared and therefore has no project allocation ceiling.
	// Keep the field for callers that still consume the existing shape.
	cpuPercent: number | null;
};

function mainService(project: Project) {
	return project.mainService?.trim() || 'app';
}

function configuredMemoryLimit(project: Project, service: string) {
	if (project.deployMode !== 'compose' || service === mainService(project)) {
		return project.memoryLimitMb;
	}

	const override = project.serviceResources?.[service];
	return override?.memoryLimitMb > 0 ? override.memoryLimitMb : DEFAULT_COMPOSE_SERVICE_MEMORY_MB;
}

/**
 * Build the memory chart ceiling from the resources assigned to the visible
 * runtime services. CPU intentionally has no allocation scale because project
 * runtimes share available host CPU instead of receiving a hard per-project cap.
 */
export function projectResourceScale(project: Project, metrics: ContainerMetrics[]): ProjectResourceScale {
	if (project.deployMode === 'static' || metrics.length === 0) {
		return { memoryMb: null, cpuPercent: null };
	}

	let memoryMb = 0;
	for (const metric of metrics) {
		const configuredMemory = configuredMemoryLimit(project, metric.service);
		const runtimeMemoryLimit = Number.isFinite(metric.memoryLimitMb) && metric.memoryLimitMb > 0
			? metric.memoryLimitMb
			: configuredMemory;
		memoryMb = Math.max(memoryMb, runtimeMemoryLimit);
	}

	return {
		memoryMb: memoryMb > 0 ? memoryMb : null,
		cpuPercent: null
	};
}

/**
 * Aggregate the hard memory allocation represented by the visible runtime
 * services. CPU remains observable live but is not an allocation.
 */
export function projectResourceAllocation(project: Project, metrics: ContainerMetrics[]): ProjectResourceScale {
	if (project.deployMode === 'static' || metrics.length === 0) {
		return { memoryMb: null, cpuPercent: null };
	}

	let memoryMb = 0;
	for (const metric of metrics) {
		const configuredMemory = configuredMemoryLimit(project, metric.service);
		const runtimeMemoryLimit = Number.isFinite(metric.memoryLimitMb) && metric.memoryLimitMb > 0
			? metric.memoryLimitMb
			: configuredMemory;
		memoryMb += runtimeMemoryLimit;
	}

	return {
		memoryMb: memoryMb > 0 ? memoryMb : null,
		cpuPercent: null
	};
}
