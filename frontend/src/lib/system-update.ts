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
	available: boolean;
};

export type UpdateSnapshot = {
	status: UpdateStatus;
	release: UpdateRelease | null;
};

export const updateSteps = [
	{ key: 'resolve', label: 'Resolve release', description: 'Resolve the published tag and target revision.' },
	{ key: 'artifacts', label: 'Check artifacts', description: 'Confirm immutable release images are available.' },
	{ key: 'preflight', label: 'Preflight', description: 'Validate migrations and the existing runtime.' },
	{ key: 'apply', label: 'Apply update', description: 'Advance the managed checkout and recreate the required services.' },
	{ key: 'verify', label: 'Verify', description: 'Verify the updated control plane before declaring success.' },
	{ key: 'complete', label: 'Complete', description: 'The requested release is installed.' }
] as const;

export function isUpdateBusy(snapshot: UpdateSnapshot | null | undefined) {
	return snapshot?.status.state === 'checking' || snapshot?.status.state === 'updating';
}

export function phaseStepIndex(phase: UpdatePhase) {
	switch (phase) {
		case 'resolving_release':
		case 'validating_target':
			return 0;
		case 'checking_images':
			return 1;
		case 'preflight':
			return 2;
		case 'applying':
		case 'rolling_back':
			return 3;
		case 'verifying':
			return 4;
		case 'complete':
			return 5;
		default:
			return 0;
	}
}
