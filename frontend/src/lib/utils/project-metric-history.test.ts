import { describe, expect, it } from 'vitest';

import { appendProjectMetricHistory } from './project-metric-history';

const metric = (service: string, cpu: number, memoryMb: number, memoryLimitMb = 100) => ({
	service,
	cpu,
	memoryMb,
	memoryLimitMb,
	uptime: '1m'
});

describe('appendProjectMetricHistory', () => {
	it('retains repeated samples instead of only value changes', () => {
		let history = appendProjectMetricHistory({}, [metric('app', 2, 10)], 3);
		history = appendProjectMetricHistory(history, [metric('app', 2, 10)], 3);
		expect(history.app.cpu).toEqual([2, 2]);
		expect(history.app.memoryPercent).toEqual([10, 10]);
	});

	it('keeps independent bounded history per service', () => {
		let history = appendProjectMetricHistory({}, [metric('api', 1, 10), metric('worker', 4, 40)], 2);
		history = appendProjectMetricHistory(history, [metric('api', 2, 20)], 2);
		history = appendProjectMetricHistory(history, [metric('api', 3, 30)], 2);
		expect(history.api.cpu).toEqual([2, 3]);
		expect(history.worker.cpu).toEqual([4]);
	});
});
