import type {
	Project,
	Deployment,
	DeployModeDetection,
	EnvVar,
	MetricsSnapshot,
	QuotaUsage,
	RepoInspection,
	User,
	AuditLog,
	ComposeResourceSummary,
	LogsResponse
} from '$types';

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
		create: (data: unknown):       Promise<Project>    => request('/projects',      { method: 'POST',   body: JSON.stringify(data) }),
		detectMode: (data: unknown):   Promise<DeployModeDetection> => request('/projects/detect-mode', { method: 'POST', body: JSON.stringify(data) }),
		inspectRepository: (data: unknown): Promise<RepoInspection> =>
			request('/projects/detect-mode', { method: 'POST', body: JSON.stringify({ ...(data as object), inspectOnly: true }) }),
		update: (id: string, d: unknown): Promise<Project> => request(`/projects/${id}`, { method: 'PATCH',  body: JSON.stringify(d) }),
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
		rollback: (id: string):                   Promise<Deployment>   => request(`/deployments/${id}/rollback`, { method: 'POST' })
	},

	env: {
		list:       (projectId: string):              Promise<EnvVar[]> => request(`/projects/${projectId}/env`),
		reveal:     (projectId: string, key: string): Promise<{ value: string }> => request(`/projects/${projectId}/env/${encodeURIComponent(key)}/reveal`),
		bulkUpdate: (projectId: string, d: unknown):  Promise<void>    => request(`/projects/${projectId}/env`, { method: 'PUT', body: JSON.stringify(d) }),
		delete:     (projectId: string, key: string): Promise<void>    => request(`/projects/${projectId}/env/${encodeURIComponent(key)}`, { method: 'DELETE' })
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
			request(`/admin/audit-logs?limit=${pageSize + (lookahead ? 1 : 0)}&offset=${page * pageSize}`)
	}
};
