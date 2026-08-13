import type { ComposeIssue, DeployModeDetection } from '$types';

const MAX_VISIBLE_NON_BLOCKING_COMPOSE_ISSUES = 3;

export function prioritizeCreateProjectDiagnostics(result: DeployModeDetection): DeployModeDetection {
	if (!result.composePlan || result.composePlan.issues.length === 0) return result;

	const blocking = result.composePlan.issues.filter((issue) => issue.severity === 'error');
	const nonBlocking = result.composePlan.issues.filter((issue) => issue.severity !== 'error');
	if (nonBlocking.length <= MAX_VISIBLE_NON_BLOCKING_COMPOSE_ISSUES) return result;

	const visibleNonBlocking = nonBlocking.slice(0, MAX_VISIBLE_NON_BLOCKING_COMPOSE_ISSUES);
	const hiddenCount = nonBlocking.length - visibleNonBlocking.length;
	const summary: ComposeIssue = {
		severity: 'info',
		code: 'ADDITIONAL_DIAGNOSTICS',
		message: `${hiddenCount} additional non-blocking Compose diagnostic${hiddenCount === 1 ? '' : 's'} hidden to keep required fixes prominent.`
	};

	return {
		...result,
		composePlan: {
			...result.composePlan,
			issues: [...blocking, ...visibleNonBlocking, summary]
		}
	};
}
