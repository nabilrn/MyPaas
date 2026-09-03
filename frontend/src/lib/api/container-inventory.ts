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
}

export interface RuntimeContainerLoadOptions {
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

// The Containers workspace is intentionally metadata-only. CPU/RAM belongs to
// project observability; keeping host inventory independent from runtime stats
// makes this page fast and predictable across Docker-compatible runtimes.
export async function loadRuntimeContainers(options: RuntimeContainerLoadOptions = {}): Promise<RuntimeContainer[]> {
	const response = await fetch('/api/admin/containers?telemetry=false', {
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
		service: row.service ?? ''
	};
}
