import "dotenv/config";
import http from "node:http";
import { connectKameleo, resolveCDPEndpoint } from "./cdp.js";
import { runPipeline } from "./pipeline.js";
import type { SolveRequest, SolveResponse } from "./types.js";

const PORT = Number(process.env.CAPTCHA_SIDECAR_PORT ?? 8787);

async function handleSolve(req: SolveRequest): Promise<SolveResponse> {
  const cdp = resolveCDPEndpoint(req);
  const { browser, page } = await connectKameleo(cdp);

  try {
    if (req.url && page.url() !== req.url) {
      await page.goto(req.url, { waitUntil: "domcontentloaded", timeout: 60_000 }).catch(() => undefined);
    }
    return await runPipeline(page, req);
  } finally {
    await browser.close().catch(() => undefined);
  }
}

const server = http.createServer(async (req, res) => {
  if (req.method === "GET" && req.url === "/health") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ ok: true }));
    return;
  }

  if (req.method !== "POST" || req.url !== "/solve") {
    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ success: false, message: "not found" }));
    return;
  }

  let body = "";
  for await (const chunk of req) body += chunk;

  let payload: SolveRequest;
  try {
    payload = JSON.parse(body || "{}") as SolveRequest;
  } catch {
    res.writeHead(400, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ success: false, message: "invalid json" }));
    return;
  }

  try {
    const result = await handleSolve(payload);
    res.writeHead(result.success ? 200 : 500, { "Content-Type": "application/json" });
    res.end(JSON.stringify(result));
  } catch (err) {
    res.writeHead(500, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ success: false, message: (err as Error).message }));
  }
});

server.listen(PORT, "127.0.0.1", () => {
  console.log(`captcha sidecar listening on http://127.0.0.1:${PORT}`);
  console.log("  POST /solve  — 3-tier sequential pipeline");
  console.log("  GET  /health — readiness probe");
});
