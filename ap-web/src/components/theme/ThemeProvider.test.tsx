import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

const providerSpy = vi.hoisted(() => vi.fn());

vi.mock("next-themes", () => ({
  ThemeProvider: ({ children, ...props }: { children: ReactNode } & Record<string, unknown>) => {
    providerSpy(props);
    return <div>{children}</div>;
  },
}));

import { ThemeProvider } from "./ThemeProvider";

describe("ThemeProvider", () => {
  it("forces the single Dracula dark theme", () => {
    render(
      <ThemeProvider>
        <span>child</span>
      </ThemeProvider>,
    );

    expect(providerSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        attribute: "class",
        forcedTheme: "dark",
        enableSystem: false,
        storageKey: "ap-web-theme",
      }),
    );
  });
});
