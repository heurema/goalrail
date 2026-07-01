import { afterEach, describe, expect, it } from "vitest";

import { getCliServerUrl, setGoalrailHostConfig } from "./host";

afterEach(() => {
  setGoalrailHostConfig({});
});

describe("getCliServerUrl", () => {
  it("returns window.location.origin when no suffix is configured", () => {
    setGoalrailHostConfig({});
    const url = getCliServerUrl();
    expect(url).toBe(window.location.origin);
  });

  it("appends the configured cliServerUrlSuffix", () => {
    setGoalrailHostConfig({ cliServerUrlSuffix: "/api/2.0/goalrail" });
    const url = getCliServerUrl();
    expect(url).toBe(`${window.location.origin}/api/2.0/goalrail`);
  });

  it("handles an empty string suffix the same as no suffix", () => {
    setGoalrailHostConfig({ cliServerUrlSuffix: "" });
    expect(getCliServerUrl()).toBe(window.location.origin);
  });
});
