import type { ContainerMetrics, MetricsSnapshot } from '$types';

export function selectPrimaryProjectMetric(
	metrics: MetricsSnapshot | null,
	mainService: string | null | undefined
): ContainerMetrics | null {
	const items = metrics?.items ?? [];
	if (items.length === 0) return null;

	const preferredService = mainService?.trim();
	if (preferredService) {
		const preferred = items.find((item) => item.service === preferredService);
		if (preferred) return preferred;
	}

	return items[0] ?? null;
}

export function runtimeServiceSummary(
	metrics: MetricsSnapshot | null,
	primaryMetric: ContainerMetrics | null
): { label: string; otherServices: number } {
	const items = metrics?.items ?? [];
	if (!primaryMetric) {
		return { label: 'No live service', otherServices: 0 };
	}

	return {
		label: primaryMetric.service || 'Runtime',
		otherServices: Math.max(0, items.filter((item) => item.service !== primaryMetric.service).length)
	};
}
