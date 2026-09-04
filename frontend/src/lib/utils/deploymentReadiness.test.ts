import { describe, expect, it } from "vitest";

import {
  deploymentReadinessSteps,
  deploymentReadinessSummary,
  lastDeploymentLogLine,
} from "./deploymentReadiness";

describe("deployment readiness diagnostics", () => {
  it("shows the current stage and completed stages for an active pipeline", () => {
    expect(deploymentReadinessSteps("starting").map((step) => step.state)).toEqual([
      "complete",
      "complete",
      "complete",
      "active",
    ]);
  });

  it("marks every stage complete after a running deployment", () => {
    expect(
      deploymentReadinessSteps("running").every(
        (step) => step.state === "complete",
      ),
    ).toBe(true);
  });

  it("uses short outcome-oriented copy", () => {
    expect(deploymentReadinessSummary("starting")).toEqual({
      title: "Starting",
      detail: "Waiting for the app to become ready.",
    });
    expect(deploymentReadinessSummary("running")).toEqual({
      title: "Deployment live",
      detail: "Your app is running.",
    });
    expect(deploymentReadinessSummary("running", false)).toEqual({
      title: "Deployment complete",
      detail: "This release is not active.",
    });
  });

  it("returns the last non-empty log line without terminal color codes", () => {
    expect(lastDeploymentLogLine("Starting\n\u001b[32mHealthy\u001b[0m\n")).toBe("Healthy");
    expect(lastDeploymentLogLine(null)).toBe("");
  });
});
