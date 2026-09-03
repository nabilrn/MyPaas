import { describe, expect, it } from 'vitest';
import settings from '../../routes/projects/[id]/settings/+page.svelte?raw';
import generalInformation from '../components/ProjectEffectiveConfiguration.svelte?raw';
import environmentSettings from '../components/ProjectEnvironmentSettings.svelte?raw';
import secretField from '../components/SecretField.svelte?raw';
import selectMenu from '../components/SelectMenu.svelte?raw';

describe('project settings product contract', () => {
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
	});

	it('keeps settings navigation and general information on one neutral surface', () => {
		expect(settings).not.toContain('bg-gray-50/35');
		expect(settings).not.toContain('dark:bg-neutral-950/40');
		expect(settings).not.toContain("'bg-gray-100 text-gray-950 dark:bg-neutral-900 dark:text-white'");
		expect(settings).not.toContain('min-h-[calc(100vh-11rem)] border-t');
		expect(generalInformation).not.toContain('bg-white');
		expect(generalInformation).not.toContain('dark:bg-neutral-950');
	});

	it('keeps common source settings simple and automatic', () => {
		expect(settings).toContain('Choose what MyPaaS deploys.');
		expect(settings).toContain('Advanced source settings');
		expect(settings).toContain('ariaLabel="Deployment branch"');
		expect(settings).toContain('ariaLabel="Base directory"');
		expect(settings).not.toContain('Repository validated on');
		expect(settings).not.toContain('Validate source');
		expect(settings).not.toContain('Repository, deployment target, and runtime-facing source configuration.');
	});

	it('uses custom resource selection and hides destructive runtime cleanup from normal settings', () => {
		expect(settings).toContain('Set how much CPU and memory this project can use.');
		expect(settings).toContain('ariaLabel="Resource profile"');
		expect(settings).toContain('Advanced resource limits');
		expect(settings).toContain('Runtime resources');
		expect(settings).not.toContain('<select');
		expect(settings).not.toContain('Reset resources');
		expect(settings).not.toContain('Check resources');
		expect(settings).not.toContain('Runtime limits and project-owned Compose resources.');
		expect(selectMenu).toContain('role="listbox"');
	});

	it('shows stored environment values as information until the user edits them', () => {
		expect(settings).toContain('Variables available to your app at runtime.');
		expect(environmentSettings).not.toContain('Encrypted at rest. Reveal only when you need to inspect a stored value.');
		expect(environmentSettings).toContain('Add variable');
		expect(environmentSettings).toContain('Import .env');
		expect(secretField).toContain('••••••••');
		expect(secretField).not.toContain('<div class="field');
	});

	it('keeps webhook and danger copy short and action-oriented', () => {
		expect(settings).toContain('Deploy when changes are pushed to GitHub.');
		expect(settings).toContain('Setup guide');
		expect(settings).not.toContain('GitHub push deployment endpoint and signing secret.');
		expect(settings).not.toContain('Configure repository push deployments.');
		expect(settings).not.toContain('API polling is slower');
		expect(settings).toContain('Permanently delete this project.');
		expect(settings).not.toContain('Destructive project operations are isolated here.');
		expect(settings).not.toContain('release ports');
	});
});
