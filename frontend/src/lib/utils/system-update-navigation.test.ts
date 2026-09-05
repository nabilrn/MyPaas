import { describe, expect, it } from 'vitest';
import generalSettings from '../../routes/admin/settings/+page.svelte?raw';
import systemUpdatePage from '../../routes/admin/update/+page.svelte?raw';
import releaseNotification from '../components/ReleaseNotification.svelte?raw';

describe('system update navigation contract', () => {
	it('queues manual updates only from the dedicated updater workspace', () => {
		expect(systemUpdatePage).toContain('api.admin.triggerUpdate');
		expect(generalSettings).not.toContain('api.admin.triggerUpdate');
		expect(generalSettings).toContain("goto('/admin/update')");
		expect(releaseNotification).not.toContain('api.admin.triggerUpdate');
		expect(releaseNotification).toContain("goto('/admin/update')");
	});
});
