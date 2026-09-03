import { describe, expect, it } from "vitest";
import adminLayout from "../../routes/admin/+layout.svelte?raw";
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

  it("uses the Project Settings geometry for the admin shell without changing excluded routes", () => {
    expect(adminLayout).toContain("lg:grid-cols-[12rem_minmax(0,1fr)]");
    expect(adminLayout).toContain("lg:border-r");
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
});
