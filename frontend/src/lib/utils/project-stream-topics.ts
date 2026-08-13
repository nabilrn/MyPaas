export type ProjectDeployMode = 'static' | 'dockerfile' | 'compose' | 'registry' | 'auto';

export function projectStreamTopics(pathname: string, projectId: string, deployMode?: ProjectDeployMode | string): string {
	const base = `/projects/${projectId}`;
	if (pathname === base || pathname === `${base}/`) {
		return deployMode === 'static' ? 'status' : 'status,metrics';
	}
	if (pathname.startsWith(`${base}/logs`)) {
		return 'status,logs,deployment';
	}
	return 'status';
}
