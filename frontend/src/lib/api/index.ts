import { validateProjectCreateInput, validateProjectUpdateInput } from '$lib/validation/project';
import type {
	Project,
	Deployment,
	DeployModeDetection,
	ComposeFileDetection,
	EnvVar,
	MetricsSnapshot,
	QuotaUsage,
	RepoInspection,
	User,
	AuditLog,
	ComposeResourceSummary,
	DBStudioColumn,
	DBStudioRowPage,
	DBStudioSchema,
	DBStudioStatus,
	DBStudioTable,
	DBStudioRowFilters,
	DBStudioWriteSession,
	LogsResponse
} from '$types';

export interface HostStorageStats {
	total_bytes: number;
	available_bytes: number;
}

export interface HostNetworkStats {
	interface: string;
	rx_bytes: number;
	tx_bytes: number;
}

export interface HostStats {
	host_ram_bytes: number;
	host_cpu_cores: number;
	allocated_ram_mb: number;
	allocated_cpu: number;
	storage: HostStorageStats | null;
	network: HostNetworkStats | null;
}

export interface MigrationStatus {
	id: string;
	status: 'preparing' | 'ready' | 'failed' | 'expired';
	downloadToken?: string;
	sizeBytes?: number;
	expiresAt?: string;
	error?: string;
}

class ApiError extends Error {
	constructor(
		public code: string,
		message: string
	) {
		super(message);
		this.name = 'ApiError';
	}
}

async function request<T>(path: string, init?: RequestInit, retryOnUnauthorized = true): Promise<T> {
	const res = await fetch(`/api${path}`, {
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		...init
	});

	if (res.status === 204) {
		return undefined as T;
	}

	const body = await res.json().catch(() => ({}));

	if (!res.ok) {
		if (res.status === 401 && retryOnUnauthorized && path !== '/auth/refresh') {
			const refreshed = await fetch('/api/auth/refresh', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include'
			});
			if (refreshed.ok) {
				return request<T>(path, init, false);
			}
		}
		throw new ApiError(body.error?.code ?? 'UNKNOWN', body.error?.message ?? 'Request failed');
	}

	return (body as { data: T }).data;
}

function keepDetectedTreeRootRelative(data: unknown, result: DeployModeDetection): DeployModeDetection {
	if (!data || typeof data !== 'object' || Array.isArray(data)) return result;
	const baseDirectory = typeof (data as { baseDirectory?: unknown }).baseDirectory === 'string'
		? ((data as { baseDirectory: string }).baseDirectory.trim().replace(/^\.\//, '').replace(/^\/+|\/+$/g, ''))
		: '';
	if (!baseDirectory || baseDirectory === '.' || result.tree.length === 0) return result;

	const baseDepth = baseDirectory.split('/').filter(Boolean).length;
	return {
		...result,
		tree: result.tree.map((entry) => {
			if (entry.path === baseDirectory || entry.path.startsWith(`${baseDirectory}/`)) return entry;
			return {
				...entry,
				path: `${baseDirectory}/${entry.path}`,
				depth: entry.depth + baseDepth
			};
		})
	};
}

export const api = {
	auth: {
		me:      (): Promise<User>    => request('/auth/me'),
		logout:  (): Promise<void>   => request('/auth/logout', { method: 'POST' }),
		refresh: (): Promise<void>   => request('/auth/refresh', { method: 'POST' })
	},

	me: {
		quota: (): Promise<QuotaUsage> => request('/me/quota')
	},

	projects: {
		list:   ():                    Promise<Project[]>  => request('/projects'),
		get:    (id: string):          Promise<Project>    => request(`/projects/${id}`),
		create: (data: unknown): Promise<Project> => {
			validateProjectCreateInput(data);
			return request('/projects', { method: 'POST', body: JSON.stringify(data) });
		},
		detectMode: async (data: unknown): Promise<DeployModeDetection> => {
			const result = await request<DeployModeDetection>('/projects/detect-mode', { method: 'POST', body: JSON.stringify(data) });
			return keepDetectedTreeRootRelative(data, result);
		},
		detectCompose: (data: unknown): Promise<ComposeFileDetection> =>
			request('/projects/detect-compose', { method: 'POST', body: JSON.stringify(data) }),
		inspectRepository: (data: unknown): Promise<RepoInspection> =>
			request('/projects/detect-mode', { method: 'POST', body: JSON.stringify({ ...(data as object), inspectOnly: true }) }),
		update: (id: string, data: unknown): Promise<Project> => {
			validateProjectUpdateInput(data);
			return request(`/projects/${id}`, { method: 'PATCH', body: JSON.stringify(data) });
		},
		delete: (id: string):          Promise<void>       => request(`/projects/${id}`, { method: 'DELETE' }),
		deploy: (id: string):          Promise<Deployment> => request(`/projects/${id}/deploy`,   { method: 'POST' }),
		start:  (id: string):          Promise<void>       => request(`/projects/${id}/start`,    { method: 'POST' }),
		stop:   (id: string):          Promise<void>       => request(`/projects/${id}/stop`,     { method: 'POST' }),
		restart:(id: string):          Promise<void>       => request(`/projects/${id}/restart`,  { method: 'POST' }),
		composeResources: (id: string): Promise<ComposeResourceSummary> => request(`/projects/${id}/compose-resources`),
		resetComposeResources: (id: string): Promise<void> => request(`/projects/${id}/compose-resources/reset`, { method: 'POST' }),
		regenerateWebhookSecret: (id: string): Promise<{ webhookSecret: string }> =>
			request(`/projects/${id}/webhook-secret/regenerate`, { method: 'POST' })
	},

	deployments: {
		list:     (projectId: string, page = 0, pageSize = 20, lookahead = false): Promise<Deployment[]> =>
			request(`/projects/${projectId}/deployments?limit=${pageSize + (lookahead ? 1 : 0)}&offset=${page * pageSize}`),
		get:      (id: string):                   Promise<Deployment>   => request(`/deployments/${id}`),
		rollback: (id: string):                  Promise<Deployment>   => request(`/deployments/${id}/rollback`, { method: 'POST' })
	},

	env: {
		list:       (projectId: string):              Promise<EnvVar[]> => request(`/projects/${projectId}/env`),
		reveal:     (projectId: string, key: string): Promise<{ value: string }> => request(`/projects/${projectId}/env/${encodeURIComponent(key)}/reveal`),
		bulkUpdate: (projectId: string, d: unknown):  Promise<void>    => request(`/projects/${projectId}/env`, { method: 'PUT', body: JSON.stringify(d) }),
		delete:     (projectId: string, key: string): Promise<void>    => request(`/projects/${projectId}/env/${encodeURIComponent(key)}`, { method: 'DELETE' })
	},

	dbStudio: {
		status: (projectId: string): Promise<DBStudioStatus> => request(`/projects/${projectId}/db/status`),
		startWriteSession: (projectId: string, ttlMinutes = 15): Promise<DBStudioWriteSession> =>
			request(`/projects/${projectId}/db/write-session`, { method: 'POST', body: JSON.stringify({ ttlMinutes }) }),
		revokeWriteSession: (projectId: string, sessionId: string): Promise<void> =>
			request(`/projects/${projectId}/db/write-session/${sessionId}`, { method: 'DELETE' }),
		schemas: (projectId: string): Promise<DBStudioSchema[]> => request(`/projects/${projectId}/db/schemas`),
		tables: (projectId: string, schema: string): Promise<DBStudioTable[]> =>
			request(`/projects/${projectId}/db/tables?schema=${encodeURIComponent(schema)}`),
		columns: (projectId: string, schema: string, table: string): Promise<DBStudioColumn[]> =>
			request(`/projects/${projectId}/db/columns?schema=${encodeURIComponent(schema)}&table=${encodeURIComponent(table)}`),
		rows: (projectId: string, schema: string, table: string, limit = 100, offset = 0, filters: DBStudioRowFilters = {}): Promise<DBStudioRowPage> => {
			const params = new URLSearchParams({
				schema,
				table,
				limit: String(limit),
				offset: String(offset)
			});
			if (filters.search?.trim()) {
				params.set('search', filters.search.trim());
			}
			for (const [column, value] of Object.entries(filters.enumFilters ?? {})) {
				if (value) params.set(`filter[${column}]`, value);
			}
			return request(`/projects/${projectId}/db/rows?${params.toString()}`);
		},
		insert: (projectId: string, data: unknown): Promise<void> =>
			request(`/projects/${projectId}/db/rows`, { method: 'POST', body: JSON.stringify(data) }),
		update: (projectId: string, data: unknown): Promise<void> =>
			request(`/projects/${projectId}/db/rows`, { method: 'PATCH', body: JSON.stringify(data) }),
		delete: (projectId: string, data: unknown): Promise<void> =>
			request(`/projects/${projectId}/db/rows`, { method: 'DELETE', body: JSON.stringify(data) })
	},

	logs: {
		list: (projectId: string, tail = 500): Promise<LogsResponse> => request(`/projects/${projectId}/logs?tail=${tail}`)
	},

	metrics: {
		snapshot: (projectId: string): Promise<MetricsSnapshot> => request(`/projects/${projectId}/metrics`)
	},

	admin: {
		listUsers:   ():                       Promise<User[]> => request('/admin/users'),
		addUser:     (d: unknown):             Promise<User>   => request('/admin/users',      { method: 'POST',   body: JSON.stringify(d) }),
		removeUser:  (id: string):             Promise<void>   => request(`/admin/users/${id}`, { method: 'DELETE' }),
		listAuditLogs: (page = 0, pageSize = 50, lookahead = false): Promise<AuditLog[]> =>
			request(`/admin/audit-logs?limit=${pageSize + (lookahead ? 1 : 0)}&offset=${page * pageSize}`),
		getSettings: (): Promise<Record<string, number> & { mcp_api_token?: string; cloudflare_configured?: boolean }> => request('/admin/settings'),
		updateSettings: (d: Record<string, number>): Promise<Record<string, number>> =>
			request('/admin/settings', { method: 'PUT', body: JSON.stringify(d) }),
		updateCloudflareConfig: (token: string, zone_id: string): Promise<void> =>
			request('/admin/settings/cloudflare', { method: 'POST', body: JSON.stringify({ token, zone_id }) }),
		regenerateMCPToken: (): Promise<Record<string, number> & { mcp_api_token?: string }> =>
			request('/admin/settings/mcp-token/regenerate', { method: 'POST' }),
		prepareMigration: (): Promise<MigrationStatus> =>
			request('/admin/migrate/prepare', { method: 'POST' }),
		migrationStatus: (id: string): Promise<MigrationStatus> =>
			request(`/admin/migrate/${id}/status`),
		getHostStats: (): Promise<HostStats> => request('/admin/host-stats')
	}
};