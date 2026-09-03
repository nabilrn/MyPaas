export type AdministrationNavKey =
  | "general"
  | "users"
  | "backup"
  | "migration"
  | "mcp"
  | "audit-logs";

export type AdministrationNavItem = {
  key: AdministrationNavKey;
  href: string;
  label: string;
};

export type AdministrationNavGroup = {
  label: string;
  items: readonly AdministrationNavItem[];
};

export const administrationNavGroups: readonly AdministrationNavGroup[] = [
  {
    label: "Platform",
    items: [
      { key: "general", href: "/admin/settings", label: "General" },
      { key: "users", href: "/admin/users", label: "Users" },
    ],
  },
  {
    label: "Operations",
    items: [
      { key: "backup", href: "/admin/backup", label: "Backup" },
      { key: "migration", href: "/admin/migration", label: "Migration" },
    ],
  },
  {
    label: "Integrations",
    items: [{ key: "mcp", href: "/admin/mcp", label: "MCP" }],
  },
  {
    label: "Activity",
    items: [
      { key: "audit-logs", href: "/admin/audit-logs", label: "Audit logs" },
    ],
  },
] as const;

export const administrationNavItems: readonly AdministrationNavItem[] =
  administrationNavGroups.flatMap((group) => group.items);

export function isAdministrationPath(pathname: string): boolean {
  return pathname === "/admin" || pathname.startsWith("/admin/");
}

export function isAdministrationNavItemActive(
  item: AdministrationNavItem,
  pathname: string,
): boolean {
  return pathname === item.href || pathname.startsWith(`${item.href}/`);
}

export function administrationNavItemForPath(
  pathname: string,
): AdministrationNavItem {
  return (
    administrationNavItems.find((item) =>
      isAdministrationNavItemActive(item, pathname),
    ) ?? administrationNavItems[0]
  );
}
