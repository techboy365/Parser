import type { Page } from "playwright";
import { waitUntilClear } from "../detect.js";
import { extractRecaptchaParams, injectRecaptchaToken } from "../helpers/recaptcha.js";
import { sleep, type TierResult } from "../types.js";

const TWO_CAPTCHA_IN = "https://2captcha.com/in.php";
const TWO_CAPTCHA_RES = "https://2captcha.com/res.php";

/**
 * Tier 3 — 2Captcha provider fallback with Google data-s support.
 */
export async function runProviderTier(page: Page): Promise<TierResult> {
  const apiKey = process.env.TWOCAPTCHA_API_KEY ?? "";
  if (!apiKey) {
    return { ok: false, tier: "provider", message: "TWOCAPTCHA_API_KEY not set" };
  }

  try {
    const params = await extractRecaptchaParams(page);
    if (!params) {
      return { ok: false, tier: "provider", message: "could not extract recaptcha params" };
    }

    const body = new URLSearchParams({
      key: apiKey,
      method: "userrecaptcha",
      googlekey: params.sitekey,
      pageurl: params.pageurl,
      json: "1",
    });
    if (params.datas) {
      body.set("data-s", params.datas);
    }

    const createResp = await fetch(TWO_CAPTCHA_IN, { method: "POST", body });
    const createJson = (await createResp.json()) as { status: number; request: string };
    if (createJson.status !== 1) {
      return { ok: false, tier: "provider", message: `2captcha create failed: ${createJson.request}` };
    }

    const taskId = createJson.request;
    const deadline = Date.now() + 120_000;
    let token = "";

    while (Date.now() < deadline) {
      await sleep(5000);
      const pollURL = `${TWO_CAPTCHA_RES}?key=${encodeURIComponent(apiKey)}&action=get&id=${encodeURIComponent(taskId)}&json=1`;
      const pollResp = await fetch(pollURL);
      const pollJson = (await pollResp.json()) as { status: number; request: string };
      if (pollJson.status === 1) {
        token = pollJson.request;
        break;
      }
      if (pollJson.request !== "CAPCHA_NOT_READY") {
        return { ok: false, tier: "provider", message: `2captcha poll failed: ${pollJson.request}` };
      }
    }

    if (!token) {
      return { ok: false, tier: "provider", message: "2captcha timed out waiting for token" };
    }

    await injectRecaptchaToken(page, token);
    await sleep(2000);

    const submit = page.locator("form button, form input[type='submit']").first();
    if (await submit.count()) {
      await submit.click({ timeout: 3000 }).catch(() => undefined);
      await sleep(2000);
    }

    const cleared = await waitUntilClear(page, 10_000);
    if (cleared) {
      return { ok: true, tier: "provider", message: "2captcha token injected and challenge cleared" };
    }
    return { ok: false, tier: "provider", message: "2captcha token injected but challenge remains" };
  } catch (err) {
    return { ok: false, tier: "provider", message: `provider tier error: ${(err as Error).message}` };
  }
}
