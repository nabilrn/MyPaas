import type { DeployStatus } from "$types";

export type DeploymentStepKey = "queued" | "cloning" | "building" | "starting";
export type DeploymentStepState = "pending" | "active" | "complete";

export interface DeploymentStep {
  key: DeploymentStepKey;
  label: string;
  state: DeploymentStepState;
}

export interface DeploymentReadinessSummary {
  title: string;
  detail: string;
}

const steps: Array<{ key: DeploymentStepKey; label: string }> = [
  { key: "queued", label: "Queued" },
  { key: "cloning", label: "Source" },
  { key: "building", label: "Build" },
  { key: "starting", label: "Readiness" },
];

const activeStepDetails: Record<DeployStatus, DeploymentReadinessSummary> = {
  queued: {
    title: "Waiting to deploy",
    detail: "Queued until a deploy slot is free.",
  },
  cloning: {
    title: "Fetching source",
    detail: "Cloning the selected branch.",
  },
  building: {
    title: "Building",
    detail: "Creating the release.",
  },
  starting: {
    title: "Starting",
    detail: "Waiting for the app to become ready.",
  },
  running: {
    title: "Deployment live",
    detail: "Your app is running.",
  },
  failed: {
    title: "Deployment failed",
    detail: "Check the last output lines for the error.",
  },
  stopped: {
    title: "Deployment stopped",
    detail: "The app is not running.",
  },
  rolled_back: {
    title: "Rolled back",
    detail: "An earlier release is active.",
  },
};

export function deploymentReadinessSteps(
  status: DeployStatus,
): DeploymentStep[] {
  const activeIndex = steps.findIndex((step) => step.key === status);
  return steps.map((step, index) => ({
    ...step,
    state:
      status === "running" || (activeIndex >= 0 && index < activeIndex)
        ? "complete"
        : index === activeIndex
          ? "active"
          : "pending",
  }));
}

export function deploymentReadinessSummary(
  status: DeployStatus,
  active = true,
): DeploymentReadinessSummary {
  if (status === "running" && !active) {
    return {
      title: "Deployment complete",
      detail: "This release is not active.",
    };
  }
  return activeStepDetails[status];
}

export function lastDeploymentLogLine(buildLog: string | null): string {
  if (!buildLog) return "";
  const lines = buildLog
    .split(/\r?\n/)
    .map((line) => line.replace(/\x1b\[[0-?]*[ -\/]*[@-~]/g, "").trim())
    .filter(Boolean);
  const line = lines.at(-1) ?? "";
  return line.length > 240 ? `${line.slice(0, 237)}...` : line;
}
