import type { Page, Frame, Browser, BrowserContext } from "playwright";

export type Connected = {
  browser: Browser;
  context: BrowserContext;
  page: Page;
};

export async function connectKameleo(cdpEndpoint: string): Promise<Connected> {
  const { chromium } = await import("playwright");
  const browser = await chromium.connectOverCDP(cdpEndpoint, { timeout: 90_000 });
  const context = browser.contexts()[0];
  if (!context) {
    throw new Error("no browser context from kameleo CDP");
  }
  const pages = context.pages();
  const page = pages.length > 0 ? pages[0] : await context.newPage();
  return { browser, context, page };
}

export function resolveCDPEndpoint(req: { cdp_endpoint?: string; profile_id?: string; kameleo_base?: string }): string {
  if (req.cdp_endpoint) {
    return req.cdp_endpoint;
  }
  const base = (req.kameleo_base ?? process.env.KAMELEO_ENDPOINT ?? "http://localhost:5050").replace(/^http/, "ws");
  if (!req.profile_id) {
    throw new Error("profile_id or cdp_endpoint required");
  }
  return `${base}/playwright/${req.profile_id}`;
}

export async function challengeFrame(page: Page): Promise<Frame | null> {
  for (const frame of page.frames()) {
    const url = frame.url();
    if (url.includes("recaptcha") && url.includes("bframe")) {
      return frame;
    }
  }
  return null;
}

export function anchorFrame(page: Page) {
  return page.frameLocator('iframe[src*="recaptcha"][src*="anchor"]').first();
}
