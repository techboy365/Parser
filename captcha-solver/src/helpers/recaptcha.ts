import type { Page } from "playwright";

export type RecaptchaParams = {
  sitekey: string;
  datas: string;
  pageurl: string;
};

export async function extractRecaptchaParams(page: Page): Promise<RecaptchaParams | null> {
  const data = await page.evaluate(() => {
    const out: { sitekey?: string; datas?: string; pageurl: string } = {
      pageurl: window.location.href,
    };

    const el = document.querySelector("[data-sitekey]") as HTMLElement | null;
    if (el?.getAttribute("data-sitekey")) {
      out.sitekey = el.getAttribute("data-sitekey") ?? undefined;
    }

    const scripts = Array.from(document.querySelectorAll("script")).map((s) => s.textContent ?? "");
    for (const script of scripts) {
      const sk = script.match(/sitekey['"]?\s*[:=]\s*['"]([^'"]+)['"]/i);
      if (sk?.[1]) out.sitekey = sk[1];
      const ds = script.match(/data-s['"]?\s*[:=]\s*['"]([^'"]+)['"]/i);
      if (ds?.[1]) out.datas = ds[1];
    }

    const iframe = document.querySelector('iframe[src*="recaptcha"]') as HTMLIFrameElement | null;
    if (iframe?.src) {
      const m = iframe.src.match(/[?&]k=([^&]+)/);
      if (m?.[1]) out.sitekey = decodeURIComponent(m[1]);
    }

    return out;
  });

  if (!data.sitekey) {
    return null;
  }
  return {
    sitekey: data.sitekey,
    datas: data.datas ?? "",
    pageurl: data.pageurl,
  };
}

export async function injectRecaptchaToken(page: Page, token: string): Promise<void> {
  await page.evaluate((t) => {
    const apply = (el: Element) => {
      const ta = el as HTMLTextAreaElement;
      ta.value = t;
      ta.dispatchEvent(new Event("input", { bubbles: true }));
      ta.dispatchEvent(new Event("change", { bubbles: true }));
    };

    document.querySelectorAll('[name="g-recaptcha-response"], #g-recaptcha-response').forEach(apply);

    const cfg = (window as any).___grecaptcha_cfg;
    if (cfg?.clients) {
      for (const id of Object.keys(cfg.clients)) {
        const client = cfg.clients[id];
        for (const key of Object.keys(client)) {
          try {
            const entry = client[key];
            if (entry && typeof entry.callback === "function") {
              entry.callback(t);
            }
          } catch {
            /* ignore */
          }
        }
      }
    }
  }, token);
}

export function extractToken(result: unknown): string {
  const r = result as Record<string, unknown>;
  const solution = (r.solution ?? r) as Record<string, string>;
  const token =
    solution.gRecaptchaResponse ??
    solution.token ??
    solution["cf-turnstile-response"] ??
    "";
  if (!token) {
    throw new Error(`no token in solver response: ${JSON.stringify(r)}`);
  }
  return token;
}
