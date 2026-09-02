export interface RuntimeContainer {
	id: string;
	name: string;
	image: string;
	state: string;
	status: string;
	composeProject: string;
	service: string;
	cpu: number;
	memoryMb: number;
	memoryLimitMb: number;
	metricsAvailable: boolean;
}

export interface RuntimeContainerLoadOptions {
	telemetry?: boolean;
	signal?: AbortSignal;
}

interface InventoryEnvelope {
	data?: {
		containers?: RuntimeContainer[];
	};
	error?: {
		message?: string;
	};
}

export async function loadRuntimeContainers(options: RuntimeContainerLoadOptions = {}): Promise<RuntimeContainer[]> {
	const query = options.telemetry === false ? '?telemetry=false' : '';
	const response = await fetch(`/api/admin/containers${query}`, {
		credentials: 'include',
		headers: { Accept: 'application/json' },
		signal: options.signal
	});
	const body = (await response.json().catch(() => ({}))) as InventoryEnvelope;
	if (!response.ok) {
		throw new Error(body.error?.message || 'Failed to load host container inventory');
	}
	return body.data?.containers ?? [];
}

export function mergeRuntimeContainerTelemetry(rows: RuntimeContainer[], telemetryRows: RuntimeContainer[]): RuntimeContainer[] {
	const metricsByID = new Map(telemetryRows.map((row) => [row.id, row]));
	return rows.map((row) => {
		const telemetry = metricsByID.get(row.id);
		if (!telemetry) {
			return { ...row, cpu: 0, memoryMb: 0, memoryLimitMb: 0, metricsAvailable: false };
		}
		return {
			...row,
			cpu: telemetry.cpu,
			memoryMb: telemetry.memoryMb,
			memoryLimitMb: telemetry.memoryLimitMb,
			metricsAvailable: telemetry.metricsAvailable
		};
	});
}
