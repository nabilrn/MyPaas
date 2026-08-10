const PROJECT_NAME_PATTERN = /^[a-z0-9][a-z0-9-]{1,28}[a-z0-9]$/;

export interface ProjectCreationReadinessInput {
  name: string;
  sourceType: "git" | "registry";
  sourceReady: boolean;
  deployMode: string;
  appPort: string;
  composeDisabledReason: string;
  busy: boolean;
}

export interface ProjectCreationReadiness {
  ready: boolean;
  state: "Waiting for source" | "Analyzing deployment" | "Needs configuration" | "Ready to create";
  reason: string;
}

export function suggestProjectName(source: string): string {
  let value = source.trim();
  if (!value) return "";

  value = value.replace(/[?#].*$/, "");
  value = value.replace(/\/+$/, "");

  const sshSeparator = value.lastIndexOf(":");
  const slash = value.lastIndexOf("/");
  if (slash >= 0) {
    value = value.slice(slash + 1);
  } else if (value.startsWith("git@") && sshSeparator >= 0) {
    value = value.slice(sshSeparator + 1);
  }

  value = value.replace(/\.git$/i, "");
  value = value.replace(/@sha256:[a-f0-9]+$/i, "");
  value = value.replace(/:[^:]+$/, "");

  return value
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 30)
    .replace(/-$/g, "");
}

export function projectCreationReadiness(
  input: ProjectCreationReadinessInput,
): ProjectCreationReadiness {
  const normalizedName = input.name.trim().toLowerCase();
  if (!normalizedName) {
    return { ready: false, state: "Waiting for source", reason: "Project name is required" };
  }
  if (!PROJECT_NAME_PATTERN.test(normalizedName)) {
    return {
      ready: false,
      state: "Needs configuration",
      reason: "Project name must be 3-30 characters and use only letters, numbers, or dashes",
    };
  }
  if (!input.sourceReady) {
    return {
      ready: false,
      state: "Waiting for source",
      reason: input.sourceType === "registry" ? "Container image is required" : "Repository validation is required",
    };
  }
  if (input.busy) {
    return { ready: false, state: "Analyzing deployment", reason: "MyPaas is analyzing the selected source" };
  }
  if (input.sourceType === "git" && input.deployMode === "auto") {
    return {
      ready: false,
      state: "Analyzing deployment",
      reason: "Runtime analysis must finish before this project can be created",
    };
  }
  if (input.deployMode !== "static" && !input.appPort.trim()) {
    return {
      ready: false,
      state: "Needs configuration",
      reason: input.sourceType === "registry"
        ? "Container port is required for registry images. Set it in Advanced runtime settings."
        : "Application port could not be detected. Re-analyze the repository or set an Advanced override.",
    };
  }
  if (input.composeDisabledReason) {
    return { ready: false, state: "Needs configuration", reason: input.composeDisabledReason };
  }
  return { ready: true, state: "Ready to create", reason: "" };
}

function asRecord(input: unknown): Record<string, unknown> {
  if (!input || typeof input !== "object" || Array.isArray(input)) {
    throw new Error("Project payload must be an object");
  }
  return input as Record<string, unknown>;
}

function hasOwn(record: Record<string, unknown>, key: string) {
  return Object.prototype.hasOwnProperty.call(record, key);
}

function validateName(record: Record<string, unknown>, required: boolean) {
  if (!hasOwn(record, "name")) {
    if (required) throw new Error("Project name is required");
    return;
  }
  if (typeof record.name !== "string") {
    throw new Error("Project name must be a string");
  }
  const normalized = record.name.trim().toLowerCase();
  if (!PROJECT_NAME_PATTERN.test(normalized)) {
    throw new Error(
      "Project name must be 3-30 characters, use only letters, numbers, or dashes, and start/end with a letter or number",
    );
  }
}

function validateRequiredString(
  record: Record<string, unknown>,
  key: string,
  label: string,
) {
  const value = record[key];
  if (typeof value !== "string" || !value.trim()) {
    throw new Error(`${label} is required`);
  }
}

function validateRepoRelativePathValue(value: unknown, label: string) {
  if (value === null || value === undefined || value === "") return;
  if (typeof value !== "string") {
    throw new Error(`${label} must be a repository-relative path`);
  }
  const path = value.trim();
  if (!path) return;
  if (path.includes("\0")) {
    throw new Error(`${label} cannot contain NUL characters`);
  }
  if (path.startsWith("/") || path.startsWith("\\")) {
    throw new Error(`${label} must be relative to the repository root`);
  }
  if (path.includes("\\")) {
    throw new Error(`${label} must use forward slashes`);
  }
  if (path.split("/").some((segment) => segment === "..")) {
    throw new Error(`${label} cannot contain parent-directory segments`);
  }
}

function validateRepoRelativePath(
  record: Record<string, unknown>,
  key: string,
  label: string,
) {
  if (!hasOwn(record, key)) return;
  validateRepoRelativePathValue(record[key], label);
}

function validateComposeOverridePaths(record: Record<string, unknown>) {
  if (!hasOwn(record, "composeOverridePaths")) return;
  const value = record.composeOverridePaths;
  if (value === null || value === undefined) return;
  if (!Array.isArray(value)) {
    throw new Error("Compose override paths must be an array");
  }
  for (const path of value) {
    validateRepoRelativePathValue(path, "Compose override path");
  }
}

function validateNonNegativeNumber(
  record: Record<string, unknown>,
  key: string,
  label: string,
) {
  if (!hasOwn(record, key) || record[key] === null || record[key] === undefined)
    return;
  const value = record[key];
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throw new Error(`${label} must be a finite number`);
  }
  if (value < 0) {
    throw new Error(`${label} must be zero or greater`);
  }
}

function validatePort(
  record: Record<string, unknown>,
  required: boolean,
  allowZero = false,
) {
  if (!hasOwn(record, "appPort")) {
    if (required) throw new Error("App port is required");
    return;
  }
  const value = record.appPort;
  const minimum = allowZero ? 0 : 1;
  if (
    typeof value !== "number" ||
    !Number.isInteger(value) ||
    value < minimum ||
    value > 65535
  ) {
    throw new Error(`App port must be an integer between ${minimum} and 65535`);
  }
}

export function resolveProjectAppPort(deployMode: string, value: string): number {
  if (deployMode === "static") return 80;

  const normalized = value.trim();
  if (!normalized) {
    throw new Error(
      "Application port could not be detected. Add EXPOSE to the Dockerfile, expose/ports to Compose, or set a container port override in Advanced runtime settings.",
    );
  }

  const port = Number(normalized);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error("Application port must be an integer between 1 and 65535");
  }
  return port;
}

function validateCommon(record: Record<string, unknown>) {
  validateRepoRelativePath(record, "baseDirectory", "Base directory");
  validateRepoRelativePath(
    record,
    "staticFrontendPath",
    "Static frontend path",
  );
  validateRepoRelativePath(record, "composeFilePath", "Compose file path");
  validateRepoRelativePath(
    record,
    "composeWorkdir",
    "Compose working directory",
  );
  validateComposeOverridePaths(record);
  validateNonNegativeNumber(record, "memoryLimitMb", "Memory limit");
  validateNonNegativeNumber(record, "memoryMb", "Memory limit");
  validateNonNegativeNumber(record, "cpuLimit", "CPU limit");
}

function createSourceType(
  record: Record<string, unknown>,
  deployMode: string,
): "git" | "registry" {
  if (
    !hasOwn(record, "sourceType") ||
    record.sourceType === null ||
    record.sourceType === undefined ||
    record.sourceType === ""
  ) {
    return deployMode === "image" ? "registry" : "git";
  }
  if (typeof record.sourceType !== "string") {
    throw new Error("Source type must be git or registry");
  }
  const sourceType = record.sourceType.trim().toLowerCase();
  if (sourceType !== "git" && sourceType !== "registry") {
    throw new Error("Source type must be git or registry");
  }
  return sourceType;
}

export function validateProjectCreateInput(input: unknown): void {
  const record = asRecord(input);
  validateName(record, true);
  validateCommon(record);

  const deployMode =
    typeof record.deployMode === "string"
      ? record.deployMode.trim().toLowerCase()
      : "";
  const sourceType = createSourceType(record, deployMode);

  if (sourceType === "registry") {
    validateRequiredString(record, "imageRef", "Container image");
    if (deployMode && deployMode !== "image") {
      throw new Error("Registry source requires image deploy mode");
    }
  } else {
    validateRequiredString(record, "repoUrl", "Repository URL");
    if (deployMode === "image") {
      throw new Error("Image deploy mode requires registry source");
    }
    if (hasOwn(record, "branch") && record.branch !== "") {
      validateRequiredString(record, "branch", "Branch");
    }
  }

  if (deployMode === "compose") {
    validateRequiredString(record, "mainService", "Main service");
  }
  if (deployMode !== "static") {
    validatePort(record, true);
  } else if (hasOwn(record, "appPort")) {
    validatePort(record, false, true);
  }
}

export function validateProjectUpdateInput(input: unknown): void {
  const record = asRecord(input);
  validateName(record, false);
  validateCommon(record);
  if (hasOwn(record, "appPort")) {
    validatePort(record, false, true);
  }
}
