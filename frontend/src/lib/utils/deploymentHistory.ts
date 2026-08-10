import type { DeployStatus, ProjectStatus } from '$types';

export function isPipelineActive(status: DeployStatus): boolean {
	return status === 'queued' || status === 'cloning' || status === 'building' || status === 'starting';
}

export function isCurrentDeployment(deploymentId: string, activeDeploymentId: string | null | undefined): boolean {
	return Boolean(activeDeploymentId) && deploymentId === activeDeploymentId;
}

export function deploymentHistoryLabel(
	status: DeployStatus,
	deploymentId: string,
	activeDeploymentId: string | null | undefined,
	projectStatus: ProjectStatus | null | undefined
): string | undefined {
	if (status !== 'running') return undefined;
	if (!isCurrentDeployment(deploymentId, activeDeploymentId)) return 'Succeeded';
	return projectStatus === 'running' ? 'Active' : 'Current';
}

export function canRollbackDeployment(
	status: DeployStatus,
	deploymentId: string,
	activeDeploymentId: string | null | undefined
): boolean {
	return status === 'running' && !isCurrentDeployment(deploymentId, activeDeploymentId);
}
