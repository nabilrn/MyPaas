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

interface InventoryEnvelope {
	data?: {
		containers?: RuntimeContainer[];
	};
	error?: {
		message?: string;
	};
}

export async function loadRuntimeContainers(): Promise<RuntimeContainer[]> {
	const response = await fetch('/api/admin/containers', {
		credentials: 'include',
		headers: { Accept: 'application/json' }
	});
	const body = (await response.json().catch(() => ({}))) as InventoryEnvelope;
	if (!response.ok) {
		throw new Error(body.error?.message || 'Failed to load host container inventory');
	}
	return body.data?.containers ?? [];
}

export async function removeRuntimeContainer(id: string): Promise<void> {
	const response = await fetch(`/api/admin/containers/${encodeURIComponent(id)}`, {
		method: 'DELETE',
		credentials: 'include',
		headers: { Accept: 'application/json' }
	});
	if (response.ok) return;
	const body = (await response.json().catch(() => ({}))) as InventoryEnvelope;
	throw new Error(body.error?.message || 'Failed to remove container');
}
