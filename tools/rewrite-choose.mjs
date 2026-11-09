import { execFileSync } from "node:child_process";
import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";

const root = process.argv[2];
const gofmt = process.argv[3];

function walk(dir, acc = []) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (entry === "node_modules" || entry === ".next" || entry === "vendor") continue;
    const info = statSync(full);
    if (info.isDirectory()) walk(full, acc);
    else if (entry.endsWith(".go")) acc.push(full);
  }
  return acc;
}

const pattern =
  /\t(\w+) := ([^\n{}]+)\n\tif ([^{\n]+) \{\n\t\t(\w+) = ([^\n{}]+)\n\t\}\n\treturn \1\n/g;

function rewrite(text) {
  return text.replace(pattern, (whole, name, initial, condition, assigned, value) => {
    if (assigned !== name) return whole;
    return `\treturn fn.Choose(${condition}, ${value}, ${initial})\n`;
  });
}

function ensureImport(text, file) {
  if (!text.includes("fn.Choose(") && !text.includes("fn.Do(") && !text.includes("fn.Maybe(")) return text;
  if (text.includes('"github.com/scc-platform/shared/fn"')) return text;
  const sharedLine = text.match(/^\s*"github\.com\/scc-platform\/shared\/\w+"/m);
  if (sharedLine) return text.replace(sharedLine[0], `\t"github.com/scc-platform/shared/fn"\n${sharedLine[0]}`);
  const block = text.match(/^import \(\n([\s\S]*?)\n\)/m);
  if (block) return text.replace(block[0], `import (\n\t"github.com/scc-platform/shared/fn"\n${block[1]}\n)`);
  return text.replace(/^package .*\n/m, (line) => `${line}\nimport "github.com/scc-platform/shared/fn"\n`);
}

let changed = 0;
for (const file of walk(root)) {
  const original = readFileSync(file, "utf8");
  const rewritten = ensureImport(rewrite(original), file);
  if (rewritten === original) continue;
  writeFileSync(file, rewritten, "utf8");
  try {
    execFileSync(gofmt, ["-w", file], { stdio: "ignore" });
  } catch {
    process.stdout.write(`gofmt failed: ${relative(root, file)}\n`);
  }
  changed += 1;
}
process.stdout.write(`переписано файлов: ${changed}\n`);
