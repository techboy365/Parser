import type { Page } from "playwright";
import { anchorFrame, challengeFrame } from "../cdp.js";
import { waitUntilClear } from "../detect.js";
import { sleep, type TierResult } from "../types.js";

/**
 * Tier 1 — human-like interaction in the browser (no token injection).
 * Clicks checkbox, interacts with visible challenge frame when present.
 */
export async function runScriptTier(page: Page): Promise<TierResult> {
  try {
    const anchor = anchorFrame(page);
    await anchor.locator("#recaptcha-anchor").click({ timeout: 5000 }).catch(() => undefined);
    await sleep(2500);

    const bframe = await challengeFrame(page);
    if (bframe) {
      await bframe.locator(".rc-imageselect-desc-wrapper").waitFor({ timeout: 8000 }).catch(() => undefined);
      await sleep(1500);
    }

    const submit = page.locator("#submit, button[type='submit'], input[type='submit']").first();
    if (await submit.count()) {
      await submit.click({ timeout: 3000 }).catch(() => undefined);
    }

    const cleared = await waitUntilClear(page, 12_000);
    if (cleared) {
      return { ok: true, tier: "script", message: "challenge cleared after human-like interaction" };
    }
    return { ok: false, tier: "script", message: "challenge still present after script tier" };
  } catch (err) {
    return { ok: false, tier: "script", message: `script tier error: ${(err as Error).message}` };
  }
}
