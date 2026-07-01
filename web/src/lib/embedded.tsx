// Embed-mode detection seam.
//
// `false` standalone (`main.tsx` never mounts the provider), `true` when a host
// renders web through `embed.tsx` (which wraps the tree in
// `<EmbeddedProvider>`). Use it to branch UI that only makes sense in one mode
// — e.g. hide UI that only makes sense in the standalone shell; `embed.tsx`
// wraps the tree in the shared forced Dracula dark theme.
//
// It's a context (not a module-level flag) so it's reactive, overridable in
// tests via the provider, and reads cleanly with a hook from any component.

import { createContext, type ReactNode, useContext } from "react";

const EmbeddedContext = createContext(false);

export function EmbeddedProvider({ children }: { children: ReactNode }) {
  return <EmbeddedContext.Provider value={true}>{children}</EmbeddedContext.Provider>;
}

/** True when web is rendered inside a host via `embed.tsx`. */
export function useIsEmbedded(): boolean {
  return useContext(EmbeddedContext);
}
