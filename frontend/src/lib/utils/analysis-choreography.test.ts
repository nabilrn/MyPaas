import { describe, expect, it } from 'vitest';
import { isRepositoryAnalysisBusy, remainingVisualDelay } from './analysis-choreography';

describe('remainingVisualDelay', () => {
	it('holds only the unread portion of the minimum duration', () => {
		expect(remainingVisualDelay(1_000, 400, 1_125)).toBe(275);
	});

	it('adds no delay once real work already exceeded the minimum', () => {
		expect(remainingVisualDelay(1_000, 400, 1_750)).toBe(0);
	});

	it('never returns a negative or invalid delay', () => {
		expect(remainingVisualDelay(1_000, -20, 900)).toBe(0);
		expect(remainingVisualDelay(Number.NaN, 400, 1_000)).toBe(0);
	});
});

describe('isRepositoryAnalysisBusy', () => {
	it('keeps stale repository work out of registry image readiness', () => {
		expect(isRepositoryAnalysisBusy({
			sourceType: 'registry',
			detecting: true,
			inspectingRepo: true,
			repoInspectScheduled: true,
			analysisPresentationBusy: true
		})).toBe(false);
	});

	it('reports active repository work for git sources', () => {
		expect(isRepositoryAnalysisBusy({
			sourceType: 'git',
			detecting: true,
			inspectingRepo: false,
			repoInspectScheduled: false,
			analysisPresentationBusy: false
		})).toBe(true);
	});
});