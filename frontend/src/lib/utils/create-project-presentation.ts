export interface RepositoryInspectionPresentation {
	message: string;
	detail: string;
}

export function presentRepositoryInspectionError(rawMessage: string): RepositoryInspectionPresentation {
	const raw = rawMessage.trim() || 'Failed to inspect repository';
	const normalized = raw.toLowerCase();

	let message = 'MyPaas could not inspect this repository. Check the URL and repository access, then try again.';

	if (
		normalized.includes('could not read username')
		|| normalized.includes('authentication failed')
		|| normalized.includes('permission denied')
		|| normalized.includes('access denied')
	) {
		message = 'This repository is private or inaccessible to MyPaas. Use a public repository or provide a source MyPaas can access.';
	} else if (
		normalized.includes('repository not found')
		|| normalized.includes('not found')
		|| normalized.includes('does not exist')
	) {
		message = 'Repository not found. Check the repository URL and its visibility, then try again.';
	} else if (
		normalized.includes('could not resolve host')
		|| normalized.includes('connection timed out')
		|| normalized.includes('connection refused')
		|| normalized.includes('network is unreachable')
	) {
		message = 'MyPaas could not reach the repository host. Check network access and try again.';
	}

	return {
		message,
		detail: raw === message ? '' : raw
	};
}

export function createProjectBlockingSummary(input: {
	composeBlockingMessages?: string[];
	missingRequiredEnvKeys?: string[];
}) {
	const blockers: string[] = [];
	const composeBlockingMessages = input.composeBlockingMessages ?? [];
	const missingRequiredEnvKeys = input.missingRequiredEnvKeys ?? [];

	if (composeBlockingMessages.length > 0) {
		blockers.push(composeBlockingMessages[0]);
		if (composeBlockingMessages.length > 1) {
			blockers.push(`${composeBlockingMessages.length - 1} more blocking Compose issue${composeBlockingMessages.length === 2 ? '' : 's'} in diagnostics.`);
		}
	}

	if (missingRequiredEnvKeys.length > 0) {
		const visibleKeys = missingRequiredEnvKeys.slice(0, 4);
		const suffix = missingRequiredEnvKeys.length > visibleKeys.length
			? ` +${missingRequiredEnvKeys.length - visibleKeys.length} more`
			: '';
		blockers.push(`Add required environment values: ${visibleKeys.join(', ')}${suffix}.`);
	}

	return blockers;
}
