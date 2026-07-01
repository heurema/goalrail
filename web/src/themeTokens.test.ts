import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const srcRoot = join(process.cwd(), "src");
const css = readFileSync(join(srcRoot, "index.css"), "utf8");
const monacoSetup = readFileSync(join(srcRoot, "shell/monacoSetup.ts"), "utf8");
const codeBlock = readFileSync(join(srcRoot, "components/ai-elements/code-block.tsx"), "utf8");
const embed = readFileSync(join(srcRoot, "embed.tsx"), "utf8");

function expectCssToken(name: string, value: string) {
  expect(css).toContain(`${name}: ${value};`);
}

describe("Dracula theme tokens", () => {
  it("pins app CSS to the codex-theme-v1 Dracula palette", () => {
    expectCssToken("--background", "#282a36");
    expectCssToken("--foreground", "#f8f8f2");
    expectCssToken("--brand-accent", "#ff79c6");
    expectCssToken("--status-green", "#50fa7b");
    expectCssToken("--status-red", "#ff5555");
  });

  it("uses Dracula for every Monaco theme branch", () => {
    expect(monacoSetup).toContain('const LIGHT_THEME = "dracula";');
    expect(monacoSetup).toContain('const DARK_THEME = "dracula";');
  });

  it("uses Dracula for Shiki code blocks", () => {
    expect(codeBlock).toContain('themes: ["dracula"]');
    expect(codeBlock).toContain('dark: "dracula"');
    expect(codeBlock).toContain('light: "dracula"');
  });

  it("forces the embed to the same dark theme", () => {
    expect(embed).toContain('className="dark"');
    expect(embed).toContain('forcedTheme="dark"');
  });
});
