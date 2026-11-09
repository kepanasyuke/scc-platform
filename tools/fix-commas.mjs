import { execFileSync } from "node:child_process";
import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const root = process.argv[2];
const gofmt = process.argv[3];

function walk(dir, acc = []) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (entry === "node_modules" || entry === ".next") continue;
    const info = statSync(full);
    if (info.isDirectory()) walk(full, acc);
    else if (entry.endsWith(".go")) acc.push(full);
  }
  return acc;
}

function problems(files) {
  const found = [];
  for (const file of files) {
    let out = "";
    try {
      out = execFileSync(gofmt, ["-e", file], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
    } catch (error) {
      out = `${error.stdout ?? ""}${error.stderr ?? ""}`;
    }
    for (const line of out.split(/\r?\n/)) {
      const match = /^(.*\.go):(\d+):(\d+): missing ',' before newline/.exec(line.trim());
      if (match) found.push({ file: match[1], line: Number(match[2]) });
    }
  }
  return found;
}

let total = 0;
for (let round = 0; round < 6; round += 1) {
  const found = problems(walk(root));
  if (found.length === 0) break;
  for (const item of found) {
    const lines = readFileSync(item.file, "utf8").split(/\r?\n/);
    const index = item.line - 1;
    lines[index] = `${lines[index].replace(/\s+$/, "")},`;
    writeFileSync(item.file, lines.join("\n"), "utf8");
    total += 1;
    process.stdout.write(`fixed ${item.file}:${item.line}\n`);
  }
}
process.stdout.write(`итого исправлено: ${total}\n`);
