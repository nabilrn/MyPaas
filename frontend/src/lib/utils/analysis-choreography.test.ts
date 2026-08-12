import { describe, expect, it } from 'vitest';
import { remainingVisualDelay } from './analysis-choreography';

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
