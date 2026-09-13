import type { Page } from "playwright";

export type ChallengeState = {
  present: boolean;
  reason: string;
};

export async function detectChallenge(page: Page): Promise<ChallengeState> {
  const url = page.url().toLowerCase();
  const content = await page.content();
  const lower = (content + " " + url).toLowerCase();

  if (url.includes("/sorry/") || lower.includes("unusual traffic")) {
    return { present: true, reason: "google_sorry" };
  }
  if (lower.includes("recaptcha") || lower.includes("g-recaptcha")) {
    return { present: true, reason: "recaptcha" };
  }
  if (lower.includes("turnstile") || lower.includes("cf-challenge")) {
    return { present: true, reason: "turnstile" };
  }
  if (lower.includes("captcha")) {
    return { present: true, reason: "generic" };
  }
  return { present: false, reason: "clear" };
}

export async function waitUntilClear(page: Page, timeoutMs = 8000): Promise<boolean> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const state = await detectChallenge(page);
    if (!state.present) {
      return true;
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  return !(await detectChallenge(page)).present;
}
