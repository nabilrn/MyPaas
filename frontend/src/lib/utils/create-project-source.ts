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
			setupSummary: 'Image runtime and manually configured environment settings are summarized here. Low-level overrides stay in Advanced settings.',
			sectionDescription: 'Registry images are not scanned for environment variables. Add variables manually or import a .env file if the container requires them.',
			emptyState: 'No environment variables have been added yet. Add one only if the container requires it.',
			noRequiredSummary: 'No required environment values are missing.',
			portRequirement: 'Set the port that the application listens on inside the container.'
		};
	}

	return {
		setupSummary: 'Detected deployment and environment results are summarized here. Low-level overrides stay in Advanced settings.',
		sectionDescription: 'Detected from the repository automatically. Required values are shown first.',
		emptyState: 'No environment variables detected. Add one only if your application needs it.',
		noRequiredSummary: 'Scan complete · no required values missing',
		portRequirement: 'Detection could not resolve this value automatically.'
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
