/**
 * Optional CAPTCHA sidecar for the Go parser.
 *
 * Integrates with modular solvers such as Captcha-Sonic puppeteer-solver:
 * https://github.com/Captcha-Sonic/puppeteer-solver
 *
 * This skeleton exposes a HTTP endpoint the Go parser can call via captcha.solver=sidecar.
 * Wire your preferred solver implementation in solveChallenge().
 */
import http from "node:http";

const PORT = process.env.CAPTCHA_SIDECAR_PORT || 8787;

async function solveChallenge(payload) {
  // TODO: integrate puppeteer-solver or another provider here.
  // Example payload: { url, type, profile_id }
  return {
    success: false,
    message: "sidecar solver not configured; integrate puppeteer-solver in solveChallenge()",
  };
}

const server = http.createServer(async (req, res) => {
  if (req.method !== "POST" || req.url !== "/solve") {
    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ success: false, message: "not found" }));
    return;
  }

  let body = "";
  for await (const chunk of req) body += chunk;

  let payload = {};
  try {
    payload = JSON.parse(body || "{}");
  } catch {
    res.writeHead(400, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ success: false, message: "invalid json" }));
    return;
  }

  const result = await solveChallenge(payload);
  res.writeHead(result.success ? 200 : 500, { "Content-Type": "application/json" });
  res.end(JSON.stringify(result));
});

server.listen(Number(PORT), "127.0.0.1", () => {
  console.log(`captcha sidecar listening on http://127.0.0.1:${PORT}/solve`);
});
