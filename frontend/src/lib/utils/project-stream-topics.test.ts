import { describe, expect, it } from 'vitest';
import { projectStreamTopics } from './project-stream-topics';

describe('projectStreamTopics', () => {
	it('subscribes runtime project detail to status and metrics', () => {
		expect(projectStreamTopics('/projects/p1', 'p1', 'dockerfile')).toBe('status,metrics');
	});

	it('keeps static project detail off runtime metrics', () => {
		expect(projectStreamTopics('/projects/p1', 'p1', 'static')).toBe('status');
	});

	it('subscribes logs page only to status logs and deployment', () => {
		expect(projectStreamTopics('/projects/p1/logs', 'p1', 'compose')).toBe('status,logs,deployment');
	});

	it('keeps unrelated project pages on status only', () => {
		expect(projectStreamTopics('/projects/p1/settings', 'p1', 'dockerfile')).toBe('status');
	});
});
