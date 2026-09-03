export interface RuntimeContainerNetwork {
	name: string;
	ipAddress: string;
}

export interface RuntimeContainer {
	id: string;
	name: string;
	image: string;
	state: string;
	status: string;
	uptime: string;
	health: string;
	ports: string;
	restartCount: number;
	detailsAvailable: boolean;
	networks: RuntimeContainerNetwork[];
	composeProject: string;
	service: string;
	cpu: number;
	cpuAvailable: boolean;
	memoryMb: number;
	memoryLimitMb: number;
	memoryAvailable: boolean;
	metricsAvailable: boolean;
}

export interface RuntimeContainerLoadOptions {
	telemetry?: boolean;
	signal?: AbortSignal;
}

interface InventoryEnvelope {
	data?: {
		containers?: Array<Partial<RuntimeContainer> & Pick<RuntimeContainer, 'id' | 'name'>>;
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
	return (body.data?.containers ?? []).map(normalizeRuntimeContainer);
}

function normalizeRuntimeContainer(row: Partial<RuntimeContainer> & Pick<RuntimeContainer, 'id' | 'name'>): RuntimeContainer {
	const legacyMetricsAvailable = Boolean(row.metricsAvailable);
	return {
		id: row.id,
		name: row.name,
		image: row.image ?? '',
		state: row.state ?? 'unknown',
		status: row.status ?? '',
		uptime: row.uptime ?? '',
		health: row.health ?? '',
		ports: row.ports ?? '',
		restartCount: Number.isFinite(row.restartCount) ? Number(row.restartCount) : 0,
		detailsAvailable: Boolean(row.detailsAvailable),
		networks: Array.isArray(row.networks)
			? row.networks.map((network) => ({ name: network?.name ?? '', ipAddress: network?.ipAddress ?? '' }))
			: [],
		composeProject: row.composeProject ?? '',
		service: row.service ?? '',
		cpu: Number.isFinite(row.cpu) ? Number(row.cpu) : 0,
		cpuAvailable: row.cpuAvailable === undefined ? legacyMetricsAvailable : Boolean(row.cpuAvailable),
		memoryMb: Number.isFinite(row.memoryMb) ? Number(row.memoryMb) : 0,
		memoryLimitMb: Number.isFinite(row.memoryLimitMb) ? Number(row.memoryLimitMb) : 0,
		memoryAvailable: row.memoryAvailable === undefined ? legacyMetricsAvailable : Boolean(row.memoryAvailable),
		metricsAvailable: legacyMetricsAvailable
	};
}

export function mergeRuntimeContainerTelemetry(rows: RuntimeContainer[], telemetryRows: RuntimeContainer[]): RuntimeContainer[] {
	const metricsByID = new Map(telemetryRows.map((row) => [row.id, row]));
	return rows.map((row) => {
		const telemetry = metricsByID.get(row.id);
		if (!telemetry) {
			return {
				...row,
				cpu: 0,
				cpuAvailable: false,
				memoryMb: 0,
				memoryLimitMb: 0,
				memoryAvailable: false,
				metricsAvailable: false
			};
		}
		return {
			...row,
			cpu: telemetry.cpu,
			cpuAvailable: telemetry.cpuAvailable,
			memoryMb: telemetry.memoryMb,
			memoryLimitMb: telemetry.memoryLimitMb,
			memoryAvailable: telemetry.memoryAvailable,
			metricsAvailable: telemetry.metricsAvailable
		};
	});
}
