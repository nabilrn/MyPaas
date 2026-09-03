import { describe, expect, it } from 'vitest';
import settings from '../../routes/projects/[id]/settings/+page.svelte?raw';
import generalInformation from '../components/ProjectEffectiveConfiguration.svelte?raw';

describe('project settings general information contract', () => {
	it('uses user-facing general information copy', () => {
		expect(settings).toContain('General information');
		expect(settings).toContain('Basic information about this project.');
		expect(settings).not.toContain('Project identity and effective control-plane configuration.');
		expect(generalInformation).not.toContain('Effective configuration');
		expect(generalInformation).not.toContain('control-plane');
	});

	it('renders immutable project identity as information instead of readonly fields', () => {
		expect(generalInformation).toContain('Project name');
		expect(generalInformation).toContain('Fixed after creation.');
		expect(generalInformation).toContain('Public URL');
		expect(generalInformation).toContain('Deployment type');
		expect(generalInformation).toContain('Source');
		expect(generalInformation).not.toContain('<input');
		expect(settings).not.toContain('id="pname"');
		expect(settings).not.toContain('id="publicUrl"');
	});

	it('keeps settings navigation and general information on one neutral surface', () => {
		expect(settings).not.toContain('bg-gray-50/35');
		expect(settings).not.toContain('dark:bg-neutral-950/40');
		expect(settings).not.toContain("'bg-gray-100 text-gray-950 dark:bg-neutral-900 dark:text-white'");
		expect(settings).not.toContain('min-h-[calc(100vh-11rem)] border-t');
		expect(generalInformation).not.toContain('bg-white');
		expect(generalInformation).not.toContain('dark:bg-neutral-950');
	});
});
