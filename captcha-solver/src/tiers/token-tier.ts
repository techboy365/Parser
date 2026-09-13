import type { Page } from "playwright";
import { CaptchaSonic } from "captchasonic";
import { waitUntilClear } from "../detect.js";
import { extractRecaptchaParams, extractToken, injectRecaptchaToken } from "../helpers/recaptcha.js";
import { sleep, type TierResult } from "../types.js";

/**
 * Tier 2 — CaptchaSonic token method (fast API inject).
 * Uses sitekey + Google data-s when available.
 */
export async function runTokenTier(page: Page): Promise<TierResult> {
  const apiKey = process.env.CAPTCHASONIC_API_KEY ?? "";
  if (!apiKey) {
    return { ok: false, tier: "token", message: "CAPTCHASONIC_API_KEY not set" };
  }

  try {
    const params = await extractRecaptchaParams(page);
    if (!params) {
      return { ok: false, tier: "token", message: "could not extract recaptcha sitekey/data-s" };
    }

    const client = new CaptchaSonic(apiKey, { transport: "http" });
    const payload: Record<string, string> = {
      websiteURL: params.pageurl,
      websiteKey: params.sitekey,
    };
    if (params.datas) {
      payload.dataS = params.datas;
    }

    const result = await (client as any).solveRecaptchaV2Token(payload);
    const token = extractToken(result);
    await injectRecaptchaToken(page, token);
    await sleep(2000);

    const submit = page.locator("form button, form input[type='submit'], #recaptcha-demo-submit").first();
    if (await submit.count()) {
      await submit.click({ timeout: 3000 }).catch(() => undefined);
      await sleep(2000);
    }

    const cleared = await waitUntilClear(page, 10_000);
    if (cleared) {
      return { ok: true, tier: "token", message: "token injected and challenge cleared" };
    }
    return { ok: false, tier: "token", message: "token injected but challenge remains" };
  } catch (err) {
    return { ok: false, tier: "token", message: `token tier error: ${(err as Error).message}` };
  }
}
