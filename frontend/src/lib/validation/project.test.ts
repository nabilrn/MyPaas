import { describe, expect, it } from "vitest";

import {
  projectCreationReadiness,
  resolveProjectAppPort,
  suggestProjectName,
  validateProjectCreateInput,
  validateProjectUpdateInput,
} from "./project";

function validCreate(overrides: Record<string, unknown> = {}) {
  return {
    name: "demo-app",
    sourceType: "git",
    repoUrl: "https://github.com/example/demo",
    branch: "main",
    deployMode: "dockerfile",
    appPort: 3000,
    memoryLimitMb: 256,
    cpuLimit: 0.35,
    baseDirectory: null,
    staticFrontendPath: null,
    ...overrides,
  };
}

describe("new project UX helpers", () => {
  it("suggests a safe project name from git and registry sources", () => {
    expect(suggestProjectName("https://github.com/howlil/sop-generate-app.git")).toBe("sop-generate-app");
    expect(suggestProjectName("ghcr.io/howlil/my-api:v1.4.0")).toBe("my-api");
  });

  it("does not report a git project ready while runtime mode is still auto", () => {
    expect(projectCreationReadiness({
      name: "demo-app",
      sourceType: "git",
      sourceReady: true,
      deployMode: "auto",
      appPort: "",
      composeDisabledReason: "",
      busy: false,
    })).toEqual({
      ready: false,
      state: "Analyzing deployment",
      reason: "Runtime analysis must finish before this project can be created",
    });
  });

  it("reports a detected dockerfile project ready only after its port is resolved", () => {
    expect(projectCreationReadiness({
      name: "demo-app",
      sourceType: "git",
      sourceReady: true,
      deployMode: "dockerfile",
      appPort: "",
      composeDisabledReason: "",
      busy: false,
    }).ready).toBe(false);

    expect(projectCreationReadiness({
      name: "demo-app",
      sourceType: "git",
      sourceReady: true,
      deployMode: "dockerfile",
      appPort: "3000",
      composeDisabledReason: "",
      busy: false,
    })).toEqual({
      ready: true,
      state: "Ready to create",
      reason: "",
    });
  });
});

describe("resolveProjectAppPort", () => {
  it("keeps static deployments on port 80", () => {
    expect(resolveProjectAppPort("static", "")).toBe(80);
  });

  it("returns a detected or manually overridden runtime port", () => {
    expect(resolveProjectAppPort("dockerfile", "3000")).toBe(3000);
    expect(resolveProjectAppPort("compose", "8080")).toBe(8080);
    expect(resolveProjectAppPort("image", "80")).toBe(80);
  });

  it("rejects unresolved runtime ports instead of falling back to 80", () => {
    expect(() => resolveProjectAppPort("dockerfile", "")).toThrow(
      /could not be detected/i,
    );
  });

  it("rejects invalid runtime port overrides", () => {
    expect(() => resolveProjectAppPort("image", "70000")).toThrow(
      /between 1 and 65535/i,
    );
  });
});

describe("validateProjectCreateInput", () => {
  it("accepts a backend-compatible project payload", () => {
    expect(() => validateProjectCreateInput(validCreate())).not.toThrow();
  });

  it("accepts uppercase names because the backend normalizes them", () => {
    expect(() =>
      validateProjectCreateInput(validCreate({ name: "Demo-App" })),
    ).not.toThrow();
  });

  it("rejects names that fail the backend shape", () => {
    expect(() =>
      validateProjectCreateInput(validCreate({ name: "-bad-" })),
    ).toThrow(/Project name/);
  });

  it("rejects out-of-range ports before making the request", () => {
    expect(() =>
      validateProjectCreateInput(validCreate({ appPort: 70000 })),
    ).toThrow(/App port/);
  });

  it("requires a main service for compose projects", () => {
    expect(() =>
      validateProjectCreateInput(
        validCreate({ deployMode: "compose", mainService: "" }),
      ),
    ).toThrow(/Main service/);
  });

  it("rejects repository path traversal", () => {
    expect(() =>
      validateProjectCreateInput(validCreate({ baseDirectory: "../api" })),
    ).toThrow(/parent-directory/);
  });

  it("rejects backslash-based repository paths", () => {
    expect(() =>
      validateProjectCreateInput(
        validCreate({ composeFilePath: "infra\\compose.yml" }),
      ),
    ).toThrow(/forward slashes/);
  });

  it("accepts a registry project without a repository URL", () => {
    expect(() =>
      validateProjectCreateInput(
        validCreate({
          sourceType: "registry",
          repoUrl: "",
          branch: "",
          imageRef: "ghcr.io/example/demo:latest",
          deployMode: "image",
          appPort: 80,
        }),
      ),
    ).not.toThrow();
  });

  it("infers registry source for legacy image-mode payloads", () => {
    expect(() =>
      validateProjectCreateInput(
        validCreate({
          sourceType: undefined,
          repoUrl: "",
          imageRef: "nginx:latest",
          deployMode: "image",
          appPort: 80,
        }),
      ),
    ).not.toThrow();
  });

  it("requires an image reference for registry projects", () => {
    expect(() =>
      validateProjectCreateInput(
        validCreate({
          sourceType: "registry",
          repoUrl: "",
          imageRef: "",
          deployMode: "image",
          appPort: 80,
        }),
      ),
    ).toThrow(/Container image/);
  });

  it("still requires a repository URL for git projects", () => {
    expect(() =>
      validateProjectCreateInput(validCreate({ repoUrl: "" })),
    ).toThrow(/Repository URL/);
  });

  it("rejects registry projects with a non-image deploy mode", () => {
    expect(() =>
      validateProjectCreateInput(
        validCreate({
          sourceType: "registry",
          repoUrl: "",
          imageRef: "nginx:latest",
          deployMode: "dockerfile",
        }),
      ),
    ).toThrow(/requires image deploy mode/);
  });

  it("rejects git projects using image deploy mode", () => {
    expect(() =>
      validateProjectCreateInput(
        validCreate({ deployMode: "image" }),
      ),
    ).toThrow(/requires registry source/);
  });
});

describe("validateProjectUpdateInput", () => {
  it("allows a partial update", () => {
    expect(() =>
      validateProjectUpdateInput({ branch: "release" }),
    ).not.toThrow();
  });

  it("allows appPort=0 because PATCH uses it as preserve-current", () => {
    expect(() => validateProjectUpdateInput({ appPort: 0 })).not.toThrow();
  });

  it("validates paths when they are present", () => {
    expect(() =>
      validateProjectUpdateInput({ staticFrontendPath: "/absolute" }),
    ).toThrow(/repository root/);
  });
});
