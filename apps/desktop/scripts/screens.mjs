import { _electron as electron } from "playwright";
import { mkdir, stat } from "node:fs/promises";
import { join } from "node:path";

const appDir = process.cwd();
const shotsDir = join(appDir, "screenshots");

const screens = [
  ["01-cdr-ingest", "cdr-ingest"],
  ["02-call-analytics", "call-analytics"],
  ["03-sip-probe", "sip-probe"],
  ["04-queue-acd", "queue-acd"],
  ["05-supervisor-alerts", "supervisor-alerts"],
  ["06-freeswitch-esl", "freeswitch-esl"],
  ["07-desk-phone", "desk-phone"],
  ["08-pg-doctor", "pg-doctor"]
];

await mkdir(shotsDir, { recursive: true });

const application = await electron.launch({
  args: ["."],
  cwd: appDir,
  env: { ...process.env }
});

const page = await application.firstWindow();
await page.waitForSelector(".nav__item");
await page.waitForTimeout(1200);

const title = await page.title();
const meta = await page.evaluate(() => ({
  kpis: document.querySelectorAll(".kpi").length,
  nav: document.querySelectorAll(".nav__item").length
}));
process.stdout.write(`window="${title}" nav=${meta.nav} kpis=${meta.kpis}\n`);

for (const [file, id] of screens) {
  await page.click(`.nav__item[data-id="${id}"]`);
  await page.waitForTimeout(900);
  const path = join(shotsDir, `${file}.png`);
  await page.screenshot({ path });
  const info = await stat(path);
  const label = await page.evaluate(() => document.getElementById("live-label").textContent);
  process.stdout.write(`${file}.png ${Math.round(info.size / 1024)} КБ · ${label}\n`);
}

await application.close();
process.stdout.write("готово\n");
