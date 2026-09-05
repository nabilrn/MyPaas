export type UpdateState =
	| 'idle'
	| 'checking'
	| 'updating'
	| 'succeeded'
	| 'failed'
	| 'rolled_back'
	| 'blocked';

export type UpdatePhase =
	| 'idle'
	| 'resolving_release'
	| 'validating_target'
	| 'checking_images'
	| 'preflight'
	| 'applying'
	| 'verifying'
	| 'rolling_back'
	| 'complete';

export type UpdateStatus = {
	state: UpdateState;
	phase: UpdatePhase;
	channel: 'release' | 'main' | 'unknown';
	currentSha: string;
	targetSha: string;
	targetVersion: string;
	message: string;
	updatedAt: string;
};

export type UpdateRelease = {
	tagName: string;
	name: string;
	targetSha: string;
	prerelease: boolean;
	publishedAt: string;
	htmlUrl: string;
	body: string;
	available: boolean;
};

export type UpdateSnapshot = {
	status: UpdateStatus;
	release: UpdateRelease | null;
};

export function isUpdateBusy(snapshot: UpdateSnapshot | null | undefined) {
	return snapshot?.status.state === 'checking' || snapshot?.status.state === 'updating';
}

export function updateStage(phase: UpdatePhase) {
	switch (phase) {
		case 'resolving_release':
			return { step: 1, total: 6, percent: 10, label: 'Resolving release' };
		case 'validating_target':
			return { step: 1, total: 6, percent: 20, label: 'Validating target' };
		case 'checking_images':
			return { step: 2, total: 6, percent: 35, label: 'Checking artifacts' };
		case 'preflight':
			return { step: 3, total: 6, percent: 50, label: 'Running preflight' };
		case 'applying':
			return { step: 4, total: 6, percent: 75, label: 'Applying update' };
		case 'verifying':
			return { step: 5, total: 6, percent: 90, label: 'Verifying update' };
		case 'rolling_back':
			return { step: 5, total: 6, percent: 90, label: 'Restoring previous runtime' };
		case 'complete':
			return { step: 6, total: 6, percent: 100, label: 'Complete' };
		default:
			return { step: 0, total: 6, percent: 0, label: 'Waiting' };
	}
}
