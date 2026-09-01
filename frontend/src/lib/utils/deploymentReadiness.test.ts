import { describe, expect, it } from "vitest";

import {
  deploymentReadinessSteps,
  deploymentReadinessSummary,
  lastDeploymentLogLine,
} from "./deploymentReadiness";

describe("deployment readiness diagnostics", () => {
  it("shows the current stage and completed stages for an active pipeline", () => {
    expect(
      deploymentReadinessSteps("starting").map((step) => step.state),
    ).toEqual(["complete", "complete", "complete", "active"]);
  });

  it("marks every stage complete after a running deployment", () => {
    expect(
      deploymentReadinessSteps("running").every(
        (step) => step.state === "complete",
      ),
    ).toBe(true);
  });

  it("uses readiness language for the long-running starting phase", () => {
    expect(deploymentReadinessSummary("starting")).toEqual({
      title: "Waiting for service readiness",
      detail:
        "The runtime is starting and its main service must become healthy or respond.",
    });
    expect(deploymentReadinessSummary("running", false)).toEqual({
      title: "Deployment completed",
      detail:
        "This release finished successfully and is not the selected runtime.",
    });
  });

  it("returns the last non-empty log line without terminal color codes", () => {
    expect(
      lastDeploymentLogLine("Starting\n\u001b[32mHealthy\u001b[0m\n"),
    ).toBe("Healthy");
    expect(lastDeploymentLogLine(null)).toBe("");
  });
});
