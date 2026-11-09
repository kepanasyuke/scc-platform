import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const root = process.argv[2];

function goFiles(dir, acc = []) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (entry === "node_modules" || entry === ".next") continue;
    const info = statSync(full);
    if (info.isDirectory()) goFiles(full, acc);
    else if (entry.endsWith(".go")) acc.push(full);
  }
  return acc;
}

const start = /^func firstErr\(errs \.\.\.error\) error \{$/;
let changed = 0;

for (const file of goFiles(root)) {
  const lines = readFileSync(file, "utf8").split(/\r?\n/);
  const out = [];
  let index = 0;
  while (index < lines.length) {
    if (!start.test(lines[index])) {
      out.push(lines[index]);
      index += 1;
      continue;
    }
    let depth = 1;
    index += 1;
    while (index < lines.length && depth > 0) {
      depth += (lines[index].match(/\{/g) ?? []).length;
      depth -= (lines[index].match(/\}/g) ?? []).length;
      index += 1;
    }
    out.push("func firstErr(errs ...error) error {");
    out.push("\treturn httpx.FirstError(errs...)");
    out.push("}");
    changed += 1;
  }
  const text = out.join("\n");
  if (text !== lines.join("\n")) {
    writeFileSync(file, text, "utf8");
    process.stdout.write(`${file}\n`);
  }
}
process.stdout.write(`функций замена: ${changed}\n`);
