export interface RepositoryAnalysisActivity {
	sourceType: 'git' | 'registry';
	detecting: boolean;
	inspectingRepo: boolean;
	repoInspectScheduled: boolean;
	analysisPresentationBusy: boolean;
}

export function isRepositoryAnalysisBusy(activity: RepositoryAnalysisActivity) {
	return activity.sourceType === 'git' && (
		activity.detecting
		|| activity.inspectingRepo
		|| activity.repoInspectScheduled
		|| activity.analysisPresentationBusy
	);
}

export function remainingVisualDelay(startedAtMs: number, minimumDurationMs: number, nowMs = Date.now()) {
	if (!Number.isFinite(startedAtMs) || !Number.isFinite(minimumDurationMs) || !Number.isFinite(nowMs)) return 0;
	return Math.max(0, Math.ceil(minimumDurationMs - Math.max(0, nowMs - startedAtMs)));
}