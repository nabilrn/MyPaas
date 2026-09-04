import { describe, expect, it } from 'vitest';
import {
	administrationNavigationItem,
	isPrimaryNavigationItemActive,
	primaryNavigationItems
} from './primary';

describe('primary navigation', () => {
	it('keeps every primary destination unique', () => {
		expect(new Set(primaryNavigationItems.map((item) => item.href)).size).toBe(primaryNavigationItems.length);
	});

	it('keeps project detail routes under Projects', () => {
		const projects = primaryNavigationItems.find((item) => item.key === 'projects')!;
		expect(isPrimaryNavigationItemActive(projects, '/projects')).toBe(true);
		expect(isPrimaryNavigationItemActive(projects, '/projects/123/settings/resources')).toBe(true);
		expect(isPrimaryNavigationItemActive(projects, '/containers')).toBe(false);
	});

	it('treats every administration leaf as Administration', () => {
		expect(isPrimaryNavigationItemActive(administrationNavigationItem, '/admin/settings')).toBe(true);
		expect(isPrimaryNavigationItemActive(administrationNavigationItem, '/admin/audit-logs')).toBe(true);
		expect(isPrimaryNavigationItemActive(administrationNavigationItem, '/projects')).toBe(false);
	});
});
