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
    title: "Waiting for a deploy slot",
    detail: "The deployment worker will start when host capacity is available.",
  },
  cloning: {
    title: "Reading the repository",
    detail:
      "MyPaaS is cloning the selected branch before it inspects the runtime.",
  },
  building: {
    title: "Building the workload",
    detail:
      "The image or static release is being built from the selected source.",
  },
  starting: {
    title: "Waiting for service readiness",
    detail:
      "The runtime is starting and its main service must become healthy or respond.",
  },
  running: {
    title: "Deployment is serving traffic",
    detail:
      "All deployment stages completed and the selected runtime is active.",
  },
  failed: {
    title: "Deployment failed",
    detail:
      "Review the error and the last output line to find the failing stage.",
  },
  stopped: {
    title: "Deployment is stopped",
    detail:
      "The last deployment completed, but the project runtime is not serving traffic.",
  },
  rolled_back: {
    title: "Deployment was rolled back",
    detail: "The deployment was replaced by an earlier successful release.",
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
      title: "Deployment completed",
      detail:
        "This release finished successfully and is not the selected runtime.",
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
