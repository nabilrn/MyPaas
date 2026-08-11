from pathlib import Path

SERVICE = Path("backend/internal/deployment/service.go")
CHANGELOG = Path("CHANGELOG.md")


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly one match, found {count}")
    return text.replace(old, new, 1)


service = SERVICE.read_text()

service = replace_once(
    service,
    '''\t\tif err := s.docker.StartComposeProject(ctx, composeProjectName(project.Name)); err != nil {
\t\t\treturn err
\t\t}
\t\tif err := s.addRuntimeRoute(ctx, project); err != nil {''',
    '''\t\tif err := s.docker.StartComposeProject(ctx, composeProjectName(project.Name)); err != nil {
\t\t\treturn err
\t\t}
\t\tif err := s.docker.WaitComposeServiceReady(ctx, composeProjectName(project.Name), mainService(project), 60*time.Second); err != nil {
\t\t\treturn err
\t\t}
\t\tif err := s.addRuntimeRoute(ctx, project); err != nil {''',
    "compose start readiness",
)

service = replace_once(
    service,
    '''\t\tif err := s.docker.RestartComposeProject(ctx, composeProjectName(project.Name)); err != nil {
\t\t\treturn err
\t\t}
\t\tif err := s.addRuntimeRoute(ctx, project); err != nil {''',
    '''\t\tif err := s.docker.RestartComposeProject(ctx, composeProjectName(project.Name)); err != nil {
\t\t\treturn err
\t\t}
\t\tif err := s.docker.WaitComposeServiceReady(ctx, composeProjectName(project.Name), mainService(project), 60*time.Second); err != nil {
\t\t\treturn err
\t\t}
\t\tif err := s.addRuntimeRoute(ctx, project); err != nil {''',
    "compose restart readiness",
)

write_override = '''\tif err := writeComposeOverride(layout.OverrideFile, main, s.docker.ComposePortMapping(port, project.AppPort), project.MemoryLimitMb, numericToFloat(project.CpuLimit), s.cfg.ProjectNetwork, overrideImageTag, project.ServiceResources); err != nil {
\t\treturn err
\t}
\tif err := s.docker.WriteSanitizedComposeConfigMulti(ctx, layout.WorkDir, layout.EnvFile, layout.UserFiles, layout.SanitizedFile); err != nil {
\t\treturn err
\t}'''
write_override_with_env = '''\tif err := writeComposeOverride(layout.OverrideFile, main, s.docker.ComposePortMapping(port, project.AppPort), project.MemoryLimitMb, numericToFloat(project.CpuLimit), s.cfg.ProjectNetwork, overrideImageTag, project.ServiceResources); err != nil {
\t\treturn err
\t}
\tif err := injectComposeEnvFile(layout.OverrideFile, layout.EnvFile); err != nil {
\t\treturn err
\t}
\tif err := s.docker.WriteSanitizedComposeConfigMulti(ctx, layout.WorkDir, layout.EnvFile, layout.UserFiles, layout.SanitizedFile); err != nil {
\t\treturn err
\t}'''

if service.count(write_override) != 2:
    raise SystemExit(f"compose override env injection: expected two matches, found {service.count(write_override)}")
service = service.replace(write_override, write_override_with_env, 2)

service = replace_once(
    service,
    '''\tlog("Starting compose project " + composeProjectName(project.Name) + " from " + layout.PrimaryRel)
\tif err := s.docker.ComposeUp(ctx, container.ComposeUpOptions{
\t\tProjectName:  composeProjectName(project.Name),
\t\tWorkDir:      layout.WorkDir,
\t\tComposeFiles: []string{layout.SanitizedFile},
\t\tOverrideFile: layout.OverrideFile,
\t\tEnvFile:      layout.EnvFile,
\t\tProfiles:     project.ComposeProfiles,
\t}, log); err != nil {
\t\treturn err
\t}

\tif err := s.setStatus(ctx, deploymentID, "starting"); err != nil {''',
    '''\tcomposeOpts := container.ComposeUpOptions{
\t\tProjectName:  composeProjectName(project.Name),
\t\tWorkDir:      layout.WorkDir,
\t\tComposeFiles: []string{layout.SanitizedFile},
\t\tOverrideFile: layout.OverrideFile,
\t\tEnvFile:      layout.EnvFile,
\t\tProfiles:     project.ComposeProfiles,
\t}
\tlog("Pulling remote Compose images")
\tif err := s.docker.ComposePull(ctx, composeOpts, log); err != nil {
\t\treturn err
\t}
\tlog("Starting compose project " + composeProjectName(project.Name) + " from " + layout.PrimaryRel)
\tif err := s.docker.ComposeUp(ctx, composeOpts, log); err != nil {
\t\treturn err
\t}
\tlog("Waiting for compose main service " + main + " to become ready")
\tif err := s.docker.WaitComposeServiceReady(ctx, composeProjectName(project.Name), main, 60*time.Second); err != nil {
\t\treturn err
\t}

\tif err := s.setStatus(ctx, deploymentID, "starting"); err != nil {''',
    "normal compose pull/readiness",
)

service = replace_once(
    service,
    '''\tlog("Starting compose rollback " + composeProjectName(project.Name) + " from " + layout.PrimaryRel)
\tif err := s.docker.ComposeUp(ctx, container.ComposeUpOptions{
\t\tProjectName:  composeProjectName(project.Name),
\t\tWorkDir:      layout.WorkDir,
\t\tComposeFiles: []string{layout.SanitizedFile},
\t\tOverrideFile: layout.OverrideFile,
\t\tEnvFile:      layout.EnvFile,
\t\tNoBuild:      true,
\t\tProfiles:     project.ComposeProfiles,
\t}, log); err != nil {
\t\treturn err
\t}

\tlog("Updating route " + s.host(project))''',
    '''\tlog("Starting compose rollback " + composeProjectName(project.Name) + " from " + layout.PrimaryRel)
\tif err := s.docker.ComposeUp(ctx, container.ComposeUpOptions{
\t\tProjectName:  composeProjectName(project.Name),
\t\tWorkDir:      layout.WorkDir,
\t\tComposeFiles: []string{layout.SanitizedFile},
\t\tOverrideFile: layout.OverrideFile,
\t\tEnvFile:      layout.EnvFile,
\t\tNoBuild:      true,
\t\tProfiles:     project.ComposeProfiles,
\t}, log); err != nil {
\t\treturn err
\t}
\tlog("Waiting for compose main service " + main + " to become ready")
\tif err := s.docker.WaitComposeServiceReady(ctx, composeProjectName(project.Name), main, 60*time.Second); err != nil {
\t\treturn err
\t}

\tlog("Updating route " + s.host(project))''',
    "compose rollback readiness",
)

SERVICE.write_text(service)

changelog = CHANGELOG.read_text()
changed_marker = "### Changed\n"
entry = "- Compose repository deployments now inject MyPaas project env vars into the public service, refresh remote image-only services before normal deploys, and wait for the main service to be running/healthy before routing traffic or marking the deployment running. Rollbacks keep their recorded image behavior.\n"
if entry not in changelog:
    changelog = replace_once(changelog, changed_marker, changed_marker + entry, "changelog")
    CHANGELOG.write_text(changelog)
