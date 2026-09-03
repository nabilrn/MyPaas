import { describe, expect, it } from 'vitest';
import layout from '../../routes/+layout.svelte?raw';
import databaseLayout from '../../routes/projects/[id]/database/+layout.svelte?raw';
import actionButton from '../components/ActionButton.svelte?raw';
import actionLink from '../components/ActionLink.svelte?raw';
import appHeader from '../components/AppHeader.svelte?raw';
import iconButton from '../components/IconButton.svelte?raw';
import infoDisclosure from '../components/InfoDisclosure.svelte?raw';
import projectObservability from '../components/ProjectObservability.svelte?raw';
import segmentedChoice from '../components/SegmentedChoice.svelte?raw';

describe('authenticated control geometry contract', () => {
	it('defines one desktop and coarse-pointer height for ordinary dashboard controls', () => {
		expect(layout).toContain('--control-height: 2.25rem;');
		expect(layout).toContain('--control-font-size: 0.875rem;');
		expect(layout).toContain(':global(.app-shell :is(input, select).field)');
		expect(layout).toContain(':global(.app-shell [data-action-button])');
		expect(layout).toContain(':global(.app-shell [data-action-link])');
		expect(layout).toContain(':global(.app-shell [data-icon-button])');
		expect(layout).toContain('--control-height: 2.75rem;');
	});

	it('keeps ActionButton size variants vertically and typographically identical', () => {
		expect(actionButton).toContain('h-9 min-h-9');
		expect(actionButton).toContain('rounded-md text-sm');
		expect(actionButton).toContain("xs: 'px-2.5'");
		expect(actionButton).toContain("sm: 'px-3'");
		expect(actionButton).toContain("md: 'px-4'");
		expect(actionButton).not.toMatch(/xs: '[^']*(?:h-8|text-xs)/);
		expect(actionButton).not.toMatch(/md: '[^']*h-10/);
	});

	it('keeps ActionLink size variants vertically and typographically identical', () => {
		expect(actionLink).toContain('h-9 min-h-9');
		expect(actionLink).toContain('rounded-md text-sm');
		expect(actionLink).toContain("xs: 'px-2.5'");
		expect(actionLink).toContain("sm: 'px-3'");
		expect(actionLink).toContain("md: 'px-4'");
		expect(actionLink).not.toMatch(/xs: '[^']*(?:h-8|text-xs)/);
		expect(actionLink).not.toMatch(/md: '[^']*h-10/);
	});

	it('keeps icon and custom single-line controls on the same geometry contract', () => {
		expect(iconButton).toContain('inline-flex h-9 w-9');
		expect(infoDisclosure).toContain('control-square');
		expect(appHeader).toContain('app-focus control-height flex');
		expect(databaseLayout.match(/control-height/g)?.length ?? 0).toBeGreaterThanOrEqual(2);
		expect(projectObservability).toContain('app-focus control-height flex');
	});

	it('preserves content-rich selection tiles as an explicit geometry exception', () => {
		expect(segmentedChoice).toContain('min-h-14');
	});
});
