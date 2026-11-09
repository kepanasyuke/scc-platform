import { _electron as electron } from "playwright";
import { mkdir, writeFile } from "node:fs/promises";
import { join } from "node:path";

const appDir = process.cwd();
const buildDir = join(appDir, "build");
await mkdir(buildDir, { recursive: true });

const application = await electron.launch({ args: ["."], cwd: appDir });
const page = await application.firstWindow();
await page.waitForSelector(".nav__item");

const sizes = [
  [512, "icon.png"],
  [256, "icon@256.png"]
];

for (const [size, file] of sizes) {
  const dataUrl = await page.evaluate(async (edge) => {
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#ffffff" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M5.5 4.6h2.2l1.4 3.4-1.8 1.3a10 10 0 0 0 5.1 5.1l1.3-1.8 3.4 1.4v2.2a1.8 1.8 0 0 1-1.9 1.8A14.4 14.4 0 0 1 3.7 6.5a1.8 1.8 0 0 1 1.8-1.9Z"/><path d="M15.4 4.1a6.5 6.5 0 0 1 4.4 4.4"/><path d="M14.8 7.5a3.1 3.1 0 0 1 1.9 1.9"/></svg>`;
    const canvas = document.createElement("canvas");
    canvas.width = edge;
    canvas.height = edge;
    const context = canvas.getContext("2d");
    context.fillStyle = "#ed131c";
    context.beginPath();
    context.roundRect(0, 0, edge, edge, edge * 0.22);
    context.fill();
    const image = new Image();
    await new Promise((resolve, reject) => {
      image.onload = resolve;
      image.onerror = reject;
      image.src = `data:image/svg+xml;base64,${btoa(svg)}`;
    });
    const pad = edge * 0.21;
    context.drawImage(image, pad, pad, edge - pad * 2, edge - pad * 2);
    return canvas.toDataURL("image/png");
  }, size);

  const base64 = dataUrl.replace(/^data:image\/png;base64,/, "");
  await writeFile(join(buildDir, file), Buffer.from(base64, "base64"));
  process.stdout.write(`${file} — ${size}×${size}\n`);
}

await application.close();
process.stdout.write("иконки собраны\n");
