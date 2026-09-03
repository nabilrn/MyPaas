import { describe, expect, it } from "vitest";
import adminLayout from "../../routes/admin/+layout.svelte?raw";
import adminSettingsPage from "../../routes/admin/settings/+page.svelte?raw";
import adminUsersPage from "../../routes/admin/users/+page.svelte?raw";
import adminBackupPage from "../../routes/admin/backup/+page.svelte?raw";
import adminMigrationPage from "../../routes/admin/migration/+page.svelte?raw";
import adminMcpPage from "../../routes/admin/mcp/+page.svelte?raw";
import adminAuditPage from "../../routes/admin/audit-logs/+page.svelte?raw";
import createProjectPage from "../../routes/projects/new/+page.svelte?raw";
import projectLogsPage from "../../routes/projects/[id]/logs/+page.svelte?raw";
import projectLayout from "../../routes/projects/[id]/+layout.svelte?raw";
import projectSettingsPage from "../../routes/projects/[id]/settings/+page.svelte?raw";
import adminSidebar from "../components/AdminSidebar.svelte?raw";
import appHeader from "../components/AppHeader.svelte?raw";
import navbar from "../components/Navbar.svelte?raw";
import {
  administrationNavGroups,
  administrationNavItemForPath,
  administrationNavItems,
  isAdministrationNavItemActive,
  isAdministrationPath,
} from "../navigation/administration";

describe("administration navigation contract", () => {
  it("keeps the global navigation to one Administration entry", () => {
    expect(navbar).toMatch(/label:\s*["']Administration["']/);
    expect(appHeader).toMatch(/label:\s*["']Administration["']/);
    for (const route of [
      "/admin/users",
      "/admin/audit-logs",
      "/admin/mcp",
      "/admin/backup",
      "/admin/migration",
    ]) {
      expect(navbar).not.toContain(route);
      expect(appHeader).not.toContain(route);
    }
  });

  it("marks the Administration parent active throughout the admin area", () => {
    for (const pathname of [
      "/admin",
      "/admin/settings",
      "/admin/users",
      "/admin/backup",
      "/admin/migration",
      "/admin/mcp",
      "/admin/audit-logs",
    ]) {
      expect(isAdministrationPath(pathname)).toBe(true);
      expect(
        isAdministrationNavItemActive(administrationNavItems[0], pathname),
      ).toBe(pathname === "/admin/settings");
    }
    expect(isAdministrationPath("/projects")).toBe(false);
    expect(isAdministrationPath("/administer")).toBe(false);
  });

  it("defines the shared secondary navigation groups and breadcrumbs", () => {
    expect(administrationNavGroups.map((group) => group.label)).toEqual([
      "Platform",
      "Operations",
      "Integrations",
      "Activity",
    ]);
    expect(
      administrationNavItems.map((item) => [item.label, item.href]),
    ).toEqual([
      ["General", "/admin/settings"],
      ["Users", "/admin/users"],
      ["Backup", "/admin/backup"],
      ["Migration", "/admin/migration"],
      ["MCP", "/admin/mcp"],
      ["Audit logs", "/admin/audit-logs"],
    ]);
    expect(administrationNavItemForPath("/admin/users").label).toBe("Users");
    expect(administrationNavItemForPath("/admin/audit-logs/detail").label).toBe(
      "Audit logs",
    );
    expect(appHeader).toMatch(/root:\s*["']Administration["']/);
    expect(appHeader).toMatch(/rootHref:\s*["']\/admin\/settings["']/);
    expect(appHeader).toContain("administrationNavItemForPath");
  });

  it("uses the shared admin shell without changing excluded project routes", () => {
    expect(adminLayout).toContain("lg:grid-cols-[12rem_minmax(0,1fr)]");
    expect(adminLayout).toContain("lg:border-r");
    expect(adminLayout).toContain("min-w-0 px-4 py-5 sm:px-5 lg:px-6");
    expect(adminLayout).toContain("currentSection.title");
    expect(adminLayout).toContain("currentSection.description");
    expect(adminLayout).toContain(".admin-content > .page-shell");
    expect(adminLayout).toContain("<slot />");
    expect(adminSidebar).toContain("border-l-2");
    expect(adminSidebar).toContain("bg-transparent");
    expect(adminSidebar).toContain("uppercase tracking");
    expect(adminSidebar).toContain("administrationNavGroups");
    expect(projectSettingsPage).toContain("ProjectSettingsNavItem");
    expect(projectLayout).toContain("main > .max-w-5xl");
    expect(createProjectPage).not.toContain("AdminSidebar");
    expect(projectLogsPage).not.toContain("AdminSidebar");
  });

  it("keeps administration page copy short", () => {
    expect(administrationNavItemForPath("/admin/settings")).toMatchObject({
      title: "General",
      description: "Host and platform defaults.",
    });
    expect(administrationNavItemForPath("/admin/users").description).toBe(
      "Manage owner access.",
    );
    expect(administrationNavItemForPath("/admin/mcp").title).toBe("MCP");
    expect(administrationNavItemForPath("/admin/audit-logs").description).toBe(
      "Review control-plane changes.",
    );
  });

  it("uses task-first administration content instead of nested panel narration", () => {
    expect(adminSettingsPage).not.toContain("SectionPanel");
    expect(adminSettingsPage).not.toContain("MAX_CONCURRENT_DEPLOYS");
    expect(adminSettingsPage).toContain("Update MyPaaS");
    expect(adminSettingsPage).toContain("may restart the control plane");

    expect(adminUsersPage).not.toContain('title="Owners"');
    expect(adminUsersPage).not.toContain('title="Add owner"');
    expect(adminUsersPage).toContain('role="dialog"');
    expect(adminUsersPage).not.toContain("Whitelisted access");

    expect(adminBackupPage).not.toContain("S3 automated backup");
    expect(adminBackupPage).toContain("Configure storage");
    expect(adminBackupPage).toContain('role="dialog"');

    expect(adminMigrationPage).not.toContain("Migration safety");
    expect(adminMigrationPage).toContain("What is included?");
    expect(adminMigrationPage).toContain("Prepare package");
    expect(adminMigrationPage).toContain("Running container projects pause briefly");

    expect(adminMcpPage).not.toContain("MCP access");
    expect(adminMcpPage).not.toContain("readonly");
    expect(adminMcpPage).toContain("Reveal API token");
    expect(adminMcpPage).toContain("Agent setup");

    expect(adminAuditPage).not.toContain("System event log");
    expect(adminAuditPage).not.toContain("!bg-gray-50/70");
    expect(adminAuditPage).toContain("Audit logs copied");
  });

  it("keeps administration dialogs keyboard reachable", () => {
    for (const pageSource of [adminUsersPage, adminBackupPage]) {
      expect(pageSource).toContain("trapDialogFocus");
      expect(pageSource).toContain("event.key === 'Escape'");
      expect(pageSource).toContain('tabindex="-1"');
      expect(pageSource).toContain("ReturnFocus");
    }
  });
});
