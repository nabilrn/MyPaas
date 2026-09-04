import { describe, expect, it } from 'vitest';
import newProjectLayout from '../../routes/projects/new/+layout.svelte?raw';
import newProjectPage from '../../routes/projects/new/+page.svelte?raw';
import capacityMetricChart from '../components/CapacityMetricChart.svelte?raw';

describe('create project workspace contract', () => {
	it('keeps Create Project as one mounted source-first page without a secondary sidebar or gated wizard', () => {
		expect(newProjectLayout).not.toContain('ProjectNewSidebar');
		expect(newProjectLayout).not.toContain('<aside');
		expect(newProjectLayout).not.toContain('ProjectNewWizardViewport');
		expect(newProjectPage).toContain('>Source</h2>');
		expect(newProjectPage).toContain('label="Deployment type"');
		expect(newProjectPage).toContain('>Environment</h2>');
		expect(newProjectPage).toContain('Advanced settings');
		expect(newProjectPage).toContain('type="submit"');
	});

	it('uses the full workspace with one neutral surface and structural strokes only', () => {
		expect(newProjectLayout).toContain('new-project-content min-w-0 w-full');
		expect(newProjectLayout).toContain('max-width: none');
		expect(newProjectLayout).not.toContain('64rem');
		expect(newProjectLayout).toContain("button[aria-pressed='true']");
		expect(newProjectLayout).toContain('background: transparent !important');
		expect(newProjectLayout).toContain('form > section + section');
		expect(newProjectLayout).toContain('border-top: 1px solid var(--workspace-divider)');
	});

	it('normalizes remaining dark neutral fills and hover states to the application surface', () => {
		expect(newProjectLayout).toContain("[class~='dark:bg-gray-800']");
		expect(newProjectLayout).toContain("[class~='dark:bg-gray-900']");
		expect(newProjectLayout).toContain("[class~='dark:bg-gray-950']");
		expect(newProjectLayout).toContain("[class~='dark:hover:bg-gray-900']");
		expect(newProjectLayout).toContain('background-color: var(--app-surface) !important');
	});

	it('keeps readiness and the create action visible as the long form scrolls', () => {
		expect(newProjectLayout).toContain('form > div:last-child');
		expect(newProjectLayout).toContain('position: sticky');
		expect(newProjectLayout).toContain('bottom: 0');
		expect(newProjectPage).toContain('displayCreationReadiness.state');
		expect(newProjectPage).toContain('Create project');
	});
});

describe('project resource chart polish', () => {
	it('uses a subtle luminous box grid without changing chart interaction', () => {
		expect(capacityMetricChart).toContain('style="filter: blur(1.1px);"');
		expect(capacityMetricChart).toContain('stroke-gray-200/55 dark:stroke-neutral-700/45');
		expect(capacityMetricChart).toContain('x1={chartWidth * 0.25}');
		expect(capacityMetricChart).toContain('y1={chartHeight * 0.25}');
		expect(capacityMetricChart).toContain('on:pointermove={handleChartPointer}');
	});
});
