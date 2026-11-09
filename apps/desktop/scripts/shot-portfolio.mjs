import { _electron as electron } from "playwright";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const appDir = process.cwd();
const root = join(appDir, "..", "..");
const page = join(root, "portfolio", "index.html");

const application = await electron.launch({ args: ["."], cwd: appDir });
const window = await application.firstWindow();
await window.setViewportSize({ width: 1200, height: 1400 });
await window.goto(pathToFileURL(page).href);
await window.waitForSelector(".card");
await window.waitForTimeout(700);
await window.screenshot({ path: join(root, "portfolio", "preview.png"), fullPage: true });
process.stdout.write("portfolio/preview.png готов\n");
await application.close();
