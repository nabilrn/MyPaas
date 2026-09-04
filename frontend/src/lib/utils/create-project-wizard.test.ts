import { describe, expect, it } from 'vitest';
import sidebar from '../components/ProjectNewSidebar.svelte?raw';
import viewport from '../components/ProjectNewWizardViewport.svelte?raw';
import newProjectLayout from '../../routes/projects/new/+layout.svelte?raw';
import secondaryNavItem from '../components/ProjectSecondaryNavItem.svelte?raw';
import capacityMetricChart from '../components/CapacityMetricChart.svelte?raw';

describe('create project wizard contract', () => {
	it('uses shared step state instead of scrolling to DOM headings', () => {
		expect(sidebar).toContain('createProjectWizard');
		expect(sidebar).toContain('setCreateProjectStep');
		expect(sidebar).not.toContain('scrollIntoView');
		expect(sidebar).not.toContain('querySelectorAll');
		expect(newProjectLayout).toContain('ProjectNewWizardViewport');
	});

	it('renders one wizard panel at a time while keeping the create form mounted', () => {
		expect(viewport).toContain("data-create-project-step");
		expect(viewport).toContain("section.hidden = section.dataset.createProjectStep !== $createProjectWizard.activeStep");
		expect(viewport).toContain("heading === 'Source' || heading === 'Preparing project'");
		expect(viewport).toContain("heading === 'Environment'");
		expect(viewport).toContain("heading === 'Deployment setup'");
		expect(viewport).toContain("summary.includes('Advanced settings')");
		expect(viewport).toContain("attributeFilter: ['disabled', 'aria-busy']");
	});

	it('supports disabled forward steps in the shared secondary navigation primitive', () => {
		expect(secondaryNavItem).toContain('export let disabled = false');
		expect(secondaryNavItem).toContain("aria-disabled={disabled ? 'true' : undefined}");
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
