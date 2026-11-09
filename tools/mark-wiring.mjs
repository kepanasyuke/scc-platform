import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const root = process.argv[2];

function appFiles(dir, acc = []) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (entry === "node_modules" || entry === ".next") continue;
    const info = statSync(full);
    if (info.isDirectory()) appFiles(full, acc);
    else if (entry === "app.go" && full.includes("internal")) acc.push(full);
  }
  return acc;
}

const targets = [/^func New\(/, /^func \(h \*handlers\) register\(/, /^func register\(/];
let changed = 0;

for (const file of appFiles(root)) {
  const lines = readFileSync(file, "utf8").split(/\r?\n/);
  const out = [];
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    const needs = targets.some((pattern) => pattern.test(line));
    const already = out.length > 0 && out[out.length - 1].includes("style:wide");
    if (needs && !already) out.push("//style:wide");
    out.push(line);
  }
  const text = out.join("\n");
  if (text !== lines.join("\n")) {
    writeFileSync(file, text, "utf8");
    changed += 1;
    process.stdout.write(`${file}\n`);
  }
}
process.stdout.write(`файлов размечено: ${changed}\n`);
