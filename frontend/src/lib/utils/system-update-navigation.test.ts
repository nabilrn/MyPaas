import { describe, expect, it } from 'vitest';
import generalSettings from '../../routes/admin/settings/+page.svelte?raw';
import systemUpdatePage from '../../routes/admin/update/+page.svelte?raw';
import systemUpdateRoute from '../../routes/internal/system-update/+server.ts?raw';
import releaseNotification from '../components/ReleaseNotification.svelte?raw';

describe('system update navigation contract', () => {
	it('queues manual updates only from the dedicated updater workspace', () => {
		expect(systemUpdatePage).toContain('api.admin.triggerUpdate');
		expect(generalSettings).not.toContain('api.admin.triggerUpdate');
		expect(generalSettings).toContain("goto('/admin/update')");
		expect(releaseNotification).not.toContain('api.admin.triggerUpdate');
		expect(releaseNotification).toContain("goto('/admin/update')");
	});

	it('checks release status as soon as owner access becomes enabled', () => {
		expect(releaseNotification).toContain('$: if (mounted) syncPolling(enabled)');
		expect(releaseNotification).toContain('syncPolling(isEnabled: boolean)');
		expect(releaseNotification).toContain("fetch('/internal/system-update'");
		expect(releaseNotification).toContain('data-update-banner');
		expect(releaseNotification).toContain('mypaas:update-banner-dismissed');
	});

	it('keeps stable and prerelease discovery aligned with host updater semantics', () => {
		expect(systemUpdateRoute).toContain('https://api.github.com/repos/nabilrn/MyPaas/releases/latest');
		expect(systemUpdateRoute).toContain('https://api.github.com/repos/nabilrn/MyPaas/releases?per_page=20');
		expect(systemUpdateRoute).toContain('comparison.status === \'ahead\'');
		expect(systemUpdateRoute).toContain('encodeURIComponent(currentSha)');
		expect(systemUpdateRoute).toContain('encodeURIComponent(release.tagName)');
	});

	it('shows real stage milestones with a visibly active progress affordance', () => {
		expect(systemUpdatePage).toContain('pollingFast ? 500 : 30_000');
		expect(systemUpdatePage).toContain('update-progress-fill');
		expect(systemUpdatePage).toContain('transform:scaleX(${progress.percent / 100})');
		expect(systemUpdatePage).toContain('@keyframes update-progress-pulse');
		expect(systemUpdatePage).toContain('aria-valuenow={progress.percent}');
	});
});
