import { describe, expect, it } from "vitest";

import { infoDisclosureState } from "./InfoDisclosure";

describe("InfoDisclosure", () => {
  it("starts collapsed and supports explicit open/close transitions", () => {
    const state = infoDisclosureState();
    expect(state.expanded).toBe(false);

    state.open();
    expect(state.expanded).toBe(true);

    state.close();
    expect(state.expanded).toBe(false);
  });

  it("toggles disclosure state without external dependencies", () => {
    const state = infoDisclosureState();

    state.toggle();
    expect(state.expanded).toBe(true);

    state.toggle();
    expect(state.expanded).toBe(false);
  });
});
