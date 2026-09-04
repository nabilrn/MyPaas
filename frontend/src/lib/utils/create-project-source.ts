export type CreateProjectSourceType = 'git' | 'registry';

type EnvironmentDraftLike = {
	source: string;
	defaultValue?: string;
	services?: string[];
	conflict?: unknown;
};

export function createProjectEnvironmentCopy(sourceType: CreateProjectSourceType) {
	if (sourceType === 'registry') {
		return {
			setupSummary: 'Review image settings and required environment values.',
			sectionDescription: 'Registry images are not scanned. Add variables only when the container needs them.',
			emptyState: 'No environment variables have been added.',
			noRequiredSummary: 'No required values missing.',
			portRequirement: 'Set the port the application listens on inside the container.'
		};
	}

	return {
		setupSummary: 'Review detected runtime, required values, and blockers.',
		sectionDescription: 'Detected from the repository. Required values appear first.',
		emptyState: 'No environment variables detected.',
		noRequiredSummary: 'Scan complete · no required values missing',
		portRequirement: 'Detection did not find a container port. Set it manually.'
	};
}

export function retainUserProvidedEnvironmentDrafts<T extends EnvironmentDraftLike>(drafts: T[]): T[] {
	return drafts.flatMap((draft) => {
		const retainedSources = draft.source
			.split(',')
			.map((source) => source.trim())
			.filter((source) => source === 'manual' || source === 'env-file');

		if (retainedSources.length === 0) return [];

		const {
			defaultValue: _defaultValue,
			services: _services,
			conflict: _conflict,
			...rest
		} = draft;

		return [{ ...rest, source: retainedSources.join(', ') } as T];
	});
}
