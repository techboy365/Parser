import type { Page } from "playwright";
import { waitUntilClear } from "./detect.js";
import { runProviderTier } from "./tiers/provider-tier.js";
import { runScriptTier } from "./tiers/script-tier.js";
import { runTokenTier } from "./tiers/token-tier.js";
import type { SolveRequest, SolveResponse, TierResult } from "./types.js";

/**
 * Sequential tier pipeline — one tier at a time, re-detect between attempts.
 * Tiers never run in parallel to avoid cross-interference.
 */
export async function runPipeline(page: Page, req: SolveRequest): Promise<SolveResponse> {
  const tiers: Array<{ enabled: boolean; run: () => Promise<TierResult> }> = [
    { enabled: req.enable_script !== false, run: () => runScriptTier(page) },
    { enabled: req.enable_token !== false, run: () => runTokenTier(page) },
    { enabled: req.enable_provider !== false, run: () => runProviderTier(page) },
  ];

  const attempts: TierResult[] = [];

  for (const tier of tiers) {
    if (!tier.enabled) {
      continue;
    }

    const result = await tier.run();
    attempts.push(result);

    if (result.ok) {
      const cleared = await waitUntilClear(page, 8000);
      if (cleared) {
        return {
          success: true,
          tier: result.tier,
          message: result.message,
        };
      }
    }
  }

  const summary = attempts.map((a) => `${a.tier}: ${a.message}`).join(" | ");
  return {
    success: false,
    message: summary || "no tiers enabled",
  };
}
