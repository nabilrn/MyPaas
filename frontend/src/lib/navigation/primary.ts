import { isAdministrationPath } from './administration';

export type PrimaryNavigationKey = 'projects' | 'containers' | 'shell' | 'administration';

export type PrimaryNavigationItem = {
	key: PrimaryNavigationKey;
	href: string;
	label: string;
	ownerOnly: boolean;
	section: 'workspace' | 'administration';
};

export const primaryNavigationItems: readonly PrimaryNavigationItem[] = [
	{ key: 'projects', href: '/projects', label: 'Projects', ownerOnly: false, section: 'workspace' },
	{ key: 'containers', href: '/containers', label: 'Containers', ownerOnly: false, section: 'workspace' },
	{ key: 'shell', href: '/shell', label: 'Shell', ownerOnly: true, section: 'workspace' },
	{ key: 'administration', href: '/admin/settings', label: 'Administration', ownerOnly: true, section: 'administration' }
] as const;

export const workspaceNavigationItems = primaryNavigationItems.filter((item) => item.section === 'workspace');
export const administrationNavigationItem = primaryNavigationItems.find((item) => item.section === 'administration')!;

export function isPrimaryNavigationItemActive(item: PrimaryNavigationItem, pathname: string): boolean {
	if (item.key === 'administration') return isAdministrationPath(pathname);
	if (item.key === 'projects') return pathname === '/projects' || pathname.startsWith('/projects/');
	return pathname === item.href || pathname.startsWith(`${item.href}/`);
}
