export interface MCPAPIToken {
	id: string;
	userId: string;
	name: string;
	prefix: string;
	scopes: string[];
	expiresAt?: string;
	lastUsedAt?: string;
	createdAt: string;
	revokedAt?: string;
}

export interface MCPAPITokenList {
	tokens: MCPAPIToken[];
	allowedScopes: string[];
	defaultScopes: string[];
}

export interface CreatedMCPAPIToken extends MCPAPIToken {
	token: string;
}

async function request<T>(path: string, init?: RequestInit, retryOnUnauthorized = true): Promise<T> {
	const response = await fetch(`/api${path}`, {
		credentials: 'include',
		headers: { 'Content-Type': 'application/json' },
		...init
	});
	if (response.status === 204) return undefined as T;
	const body = await response.json().catch(() => ({}));
	if (!response.ok) {
		if (response.status === 401 && retryOnUnauthorized) {
			const refreshed = await fetch('/api/auth/refresh', {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' }
			});
			if (refreshed.ok) return request<T>(path, init, false);
		}
		throw new Error(body.error?.message ?? 'Request failed');
	}
	return (body as { data: T }).data;
}

export const remoteMCPApi = {
	listTokens: (): Promise<MCPAPITokenList> => request('/admin/api-tokens'),
	createToken: (data: { name: string; scopes: string[]; expiresAt?: string }): Promise<CreatedMCPAPIToken> =>
		request('/admin/api-tokens', { method: 'POST', body: JSON.stringify(data) }),
	revokeToken: (id: string): Promise<void> => request(`/admin/api-tokens/${id}`, { method: 'DELETE' })
};
