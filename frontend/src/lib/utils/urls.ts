export function appRootDomain(hostname: string): string {
	if (hostname === 'localhost' || hostname.endsWith('.localhost') || hostname === '127.0.0.1') {
		return 'localhost';
	}
	if (hostname.startsWith('dashboard.')) {
		return hostname.slice('dashboard.'.length);
	}
	return hostname;
}

export function appScheme(protocol: string, hostname: string): string {
	const domain = appRootDomain(hostname);
	if (domain === 'localhost' || domain === '127.0.0.1') {
		return 'http';
	}
	return protocol.replace(':', '') || 'https';
}

export function projectHost(subdomain: string, hostname: string): string {
	return `${subdomain}.${appRootDomain(hostname)}`;
}

export function projectURL(subdomain: string, protocol: string, hostname: string): string {
	return `${appScheme(protocol, hostname)}://${projectHost(subdomain, hostname)}`;
}

export function projectRouteHost(subdomain: string, routeName: string, hostname: string): string {
	return `${subdomain}-${routeName}.${appRootDomain(hostname)}`;
}

export function projectRouteURL(subdomain: string, routeName: string, protocol: string, hostname: string): string {
	return `${appScheme(protocol, hostname)}://${projectRouteHost(subdomain, routeName, hostname)}`;
}

export function webhookURL(projectId: string, origin: string): string {
	return `${origin}/api/webhook/${projectId}`;
}
