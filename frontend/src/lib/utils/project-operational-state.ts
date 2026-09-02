import { isCurrentDeployment, isPipelineActive } from './deploymentHistory';
import type { Deployment, Project } from '$types';

export type ServingState = 'live' | 'offline' | 'degraded' | 'unknown';
export type ReleaseState = 'not_deployed' | 'queued' | 'deploying' | 'succeeded' | 'failed';
export type DesiredState = 'running' | 'stopped';
export type ProjectOperationalAction = 'deploy' | 'retry' | 'start' | 'view_logs' | 'view_deployment';
export type ProjectOperationalAttention = 'none' | 'info' | 'warning' | 'danger';
export type ProjectOperationalTone = 'success' | 'info' | 'warning' | 'danger' | 'neutral';
export type RuntimeEvidence = 'available' | 'unavailable' | 'not_applicable';

type OperationalProject = Pick<Project, 'status' | 'deployMode' | 'activeDeploymentId'>;
type OperationalDeployment = Pick<Deployment, 'id' | 'status'>;

export interface ProjectOperationalStateInput {
	project: OperationalProject;
	/** `undefined` means deployment history is not loaded; `null` means it is loaded and empty. */
	latestDeployment?: OperationalDeployment | null;
	runtimeEvidence?: RuntimeEvidence;
}

export interface ProjectOperationalState {
	serving: ServingState;
	release: ReleaseState;
	desired: DesiredState;
	headline: string;
	detail: string;
	primaryAction: ProjectOperationalAction;
	primaryActionLabel: string;
	attention: ProjectOperationalAttention;
	statusLabel: string;
	statusTone: ProjectOperationalTone;
}

export function deriveProjectOperationalState({
	project,
	latestDeployment,
	runtimeEvidence
}: ProjectOperationalStateInput): ProjectOperationalState {
	const desired: DesiredState = project.status === 'stopped' ? 'stopped' : 'running';
	const release = deriveReleaseState(project, latestDeployment);
	const serving = deriveServingState(project, runtimeEvidence);
	const hasActiveRelease = Boolean(project.activeDeploymentId);
	const latestFailed = latestDeployment?.status === 'failed';
	const pipelineActive = project.status === 'building' || Boolean(latestDeployment && isPipelineActive(latestDeployment.status));

	if (desired === 'stopped') {
		return state({
			serving: 'offline',
			release,
			desired,
			headline: 'Stopped',
			detail: 'Traffic is intentionally stopped for this project.',
			primaryAction: 'start',
			primaryActionLabel: 'Start',
			attention: 'none',
			statusLabel: 'Stopped',
			statusTone: 'neutral'
		});
	}

	if (project.status === 'crashed') {
		return state({
			serving: 'offline',
			release,
			desired,
			headline: 'Crashed',
			detail: 'The runtime exited unexpectedly and is not serving traffic.',
			primaryAction: 'view_logs',
			primaryActionLabel: 'View logs',
			attention: 'danger',
			statusLabel: 'Crashed',
			statusTone: 'danger'
		});
	}

	if (pipelineActive) {
		if (serving === 'live') {
			return state({
				serving,
				release,
				desired,
				headline: 'Deploying update',
				detail: 'A new deployment is in progress while the current release remains live.',
				primaryAction: 'view_deployment',
				primaryActionLabel: 'View deployment',
				attention: 'info',
				statusLabel: 'Deploying · live',
				statusTone: 'warning'
			});
		}
		if (serving === 'unknown') {
			return state({
				serving,
				release,
				desired,
				headline: 'Deploying; serving status unknown',
				detail: 'The deployment is in progress, but current runtime evidence is unavailable.',
				primaryAction: 'view_deployment',
				primaryActionLabel: 'View deployment',
				attention: 'warning',
				statusLabel: 'Deploying · unknown',
				statusTone: 'warning'
			});
		}
		return state({
			serving,
			release,
			desired,
			headline: 'Deploying',
			detail: 'The deployment pipeline is active and no release is serving traffic yet.',
			primaryAction: 'view_deployment',
			primaryActionLabel: 'View deployment',
			attention: 'info',
			statusLabel: 'Deploying',
			statusTone: 'warning'
		});
	}

	if (latestFailed) {
		if (serving === 'live') {
			return state({
				serving,
				release,
				desired,
				headline: 'Live; latest deploy failed',
				detail: 'The previous release is still serving traffic. Review the failed attempt before retrying.',
				primaryAction: 'view_deployment',
				primaryActionLabel: 'Review failure',
				attention: 'warning',
				statusLabel: 'Live · deploy failed',
				statusTone: 'warning'
			});
		}
		if (serving === 'unknown' && hasActiveRelease) {
			return state({
				serving,
				release,
				desired,
				headline: 'Latest deploy failed; serving status unknown',
				detail: 'A previous release is selected, but current runtime evidence is unavailable.',
				primaryAction: 'view_deployment',
				primaryActionLabel: 'Review failure',
				attention: 'warning',
				statusLabel: 'Unknown · deploy failed',
				statusTone: 'warning'
			});
		}
		return state({
			serving: 'offline',
			release,
			desired,
			headline: 'Deployment failed',
			detail: 'The latest deployment failed and no release is serving traffic.',
			primaryAction: 'retry',
			primaryActionLabel: 'Retry',
			attention: 'danger',
			statusLabel: 'Deploy failed',
			statusTone: 'danger'
		});
	}

	if (serving === 'unknown') {
		return state({
			serving,
			release,
			desired,
			headline: 'Status unknown',
			detail: 'Current runtime evidence is unavailable, so serving status cannot be confirmed.',
			primaryAction: 'view_logs',
			primaryActionLabel: 'View logs',
			attention: 'warning',
			statusLabel: 'Unknown',
			statusTone: 'neutral'
		});
	}

	if (serving === 'live') {
		const isStatic = project.deployMode === 'static';
		return state({
			serving,
			release,
			desired,
			headline: 'Live',
			detail: isStatic ? 'The current static release is published and serving traffic.' : 'The active release is serving traffic.',
			primaryAction: 'deploy',
			primaryActionLabel: 'Deploy again',
			attention: 'none',
			statusLabel: 'Live',
			statusTone: 'success'
		});
	}

	if (latestDeployment === null && !hasActiveRelease) {
		return state({
			serving: 'offline',
			release: 'not_deployed',
			desired,
			headline: 'Not deployed',
			detail: 'Source and configuration are saved, but no release has been published yet.',
			primaryAction: 'deploy',
			primaryActionLabel: 'Deploy',
			attention: 'none',
			statusLabel: 'Not deployed',
			statusTone: 'info'
		});
	}

	if (latestDeployment === undefined && project.status === 'pending' && !hasActiveRelease) {
		return state({
			serving: 'offline',
			release,
			desired,
			headline: 'Pending',
			detail: 'Deployment history is not available yet.',
			primaryAction: 'deploy',
			primaryActionLabel: 'Deploy',
			attention: 'none',
			statusLabel: 'Pending',
			statusTone: 'info'
		});
	}

	return state({
		serving: hasActiveRelease ? 'degraded' : 'unknown',
		release,
		desired,
		headline: 'Status unknown',
		detail: 'Project state and release evidence do not establish a reliable serving state.',
		primaryAction: hasActiveRelease ? 'view_deployment' : 'deploy',
		primaryActionLabel: hasActiveRelease ? 'View deployment' : 'Deploy',
		attention: 'warning',
		statusLabel: 'Unknown',
		statusTone: 'neutral'
	});
}

function deriveReleaseState(project: OperationalProject, latestDeployment: OperationalDeployment | null | undefined): ReleaseState {
	if (!latestDeployment) {
		if (project.status === 'building') return 'deploying';
		return project.activeDeploymentId ? 'succeeded' : 'not_deployed';
	}
	if (latestDeployment.status === 'queued') return 'queued';
	if (isPipelineActive(latestDeployment.status)) return 'deploying';
	if (latestDeployment.status === 'failed') return 'failed';
	return latestDeployment.status === 'running' || isCurrentDeployment(latestDeployment.id, project.activeDeploymentId)
		? 'succeeded'
		: project.activeDeploymentId
			? 'succeeded'
			: 'not_deployed';
}

function deriveServingState(project: OperationalProject, runtimeEvidence: RuntimeEvidence | undefined): ServingState {
	if (project.status === 'stopped' || project.status === 'crashed') return 'offline';
	if (runtimeEvidence === 'unavailable' && project.deployMode !== 'static') return 'unknown';
	if (project.status === 'running') return project.activeDeploymentId ? 'live' : 'unknown';
	if (project.status === 'building') return project.activeDeploymentId ? 'live' : 'offline';
	if (project.status === 'pending') return project.activeDeploymentId ? 'degraded' : 'offline';
	return 'unknown';
}

function state(value: ProjectOperationalState): ProjectOperationalState {
	return value;
}
