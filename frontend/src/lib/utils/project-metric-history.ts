import type { ContainerMetrics } from '$types';

export type ProjectMetricHistory = Record<string, {
	cpu: number[];
	memoryPercent: number[];
}>;

export function appendProjectMetricHistory(
	history: ProjectMetricHistory,
	items: ContainerMetrics[],
	limit = 120
): ProjectMetricHistory {
	const boundedLimit = Math.max(1, Math.floor(limit));
	const next: ProjectMetricHistory = { ...history };
	for (const item of items) {
		if (!item.service) continue;
		const previous = history[item.service] ?? { cpu: [], memoryPercent: [] };
		const cpu = Number.isFinite(item.cpu) ? Math.max(0, item.cpu) : 0;
		const memoryPercent = item.memoryLimitMb > 0 && Number.isFinite(item.memoryMb)
			? Math.max(0, (item.memoryMb / item.memoryLimitMb) * 100)
			: 0;
		next[item.service] = {
			cpu: [...previous.cpu, cpu].slice(-boundedLimit),
			memoryPercent: [...previous.memoryPercent, memoryPercent].slice(-boundedLimit)
		};
	}
	return next;
}
