import { mkdir, writeFile } from "node:fs/promises";
import { join } from "node:path";

const root = process.cwd();

const apps = [
  { dir: "01-cdr-ingest", slug: "cdr-ingest", port: 3101, title: "CDR Ingest", desc: "Приём и запись детализации вызовов" },
  { dir: "02-call-analytics", slug: "call-analytics", port: 3102, title: "Call Analytics", desc: "Аналитика нагрузки на TimescaleDB" },
  { dir: "03-sip-probe", slug: "sip-probe", port: 3103, title: "SIP Probe", desc: "Мониторинг SIP-транков" },
  { dir: "04-queue-acd", slug: "queue-acd", port: 3104, title: "Queue ACD", desc: "Распределение вызовов в очередях" },
  { dir: "05-supervisor-alerts", slug: "supervisor-alerts", port: 3105, title: "Supervisor Alerts", desc: "Оповещения супервизоров" },
  { dir: "06-freeswitch-esl", slug: "freeswitch-esl", port: 3106, title: "FreeSWITCH ESL", desc: "Консоль управления каналами" },
  { dir: "07-desk-phone-provisioning", slug: "desk-phone-provisioning", port: 3107, title: "Desk Phone", desc: "Провижининг стационарных аппаратов" },
  { dir: "08-pg-doctor", slug: "pg-doctor", port: 3108, title: "PG Doctor", desc: "Диагностика PostgreSQL" }
];

const packageJson = (app) => ({
  name: `@scc/${app.slug}`,
  version: "1.0.0",
  private: true,
  scripts: {
    dev: `next dev -p ${app.port}`,
    build: "next build",
    start: `next start -p ${app.port}`,
    typecheck: "tsc --noEmit"
  },
  dependencies: {
    "@scc/ui": "1.0.0",
    next: "^16.4.5",
    react: "^19.3.1",
    "react-dom": "^19.3.1"
  },
  devDependencies: {
    "@types/node": "^24.0.0",
    "@types/react": "^19.3.0",
    "@types/react-dom": "^19.3.0",
    typescript: "^5.9.3"
  }
});

const tsconfig = {
  extends: "../../../tsconfig.base.json",
  compilerOptions: {
    jsx: "preserve",
    plugins: [{ name: "next" }],
    paths: { "@/*": ["./*"] }
  },
  include: ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  exclude: ["node_modules"]
};

const nextConfig = `import type { NextConfig } from "next";

const config: NextConfig = {
  transpilePackages: ["@scc/ui"],
  poweredByHeader: false
};

export default config;
`;

const layout = (app) => `import type { ReactNode } from "react";
import type { Metadata } from "next";
import "@scc/ui/tokens.css";
import "@scc/ui/ui.css";
import "@scc/ui/global.css";

export const metadata: Metadata = {
  title: "${app.title} · SCC Platform",
  description: "${app.desc}"
};

export default function RootLayout({ children }: { children: ReactNode }): ReactNode {
  return (
    <html lang="ru">
      <body>{children}</body>
    </html>
  );
}
`;

const api = (app) => `const baseUrl = process.env.SCC_${app.slug.toUpperCase().replaceAll("-", "_")}_API ?? "http://localhost:${app.port + 5000}";

export async function apiGet<T>(path: string, fallback: T): Promise<{ data: T; live: boolean }> {
  try {
    const response = await fetch(\`\${baseUrl}\${path}\`, { cache: "no-store", signal: AbortSignal.timeout(2000) });
    const data = (await response.json()) as T;
    return { data, live: response.ok };
  } catch {
    return { data: fallback, live: false };
  }
}
`;

for (const app of apps) {
  const base = join(root, "projects", app.dir, "frontend");
  await mkdir(join(base, "app"), { recursive: true });
  await mkdir(join(base, "lib"), { recursive: true });
  await writeFile(join(base, "package.json"), `${JSON.stringify(packageJson(app), null, 2)}\n`, "utf8");
  await writeFile(join(base, "tsconfig.json"), `${JSON.stringify(tsconfig, null, 2)}\n`, "utf8");
  await writeFile(join(base, "next.config.ts"), nextConfig, "utf8");
  await writeFile(join(base, "app", "layout.tsx"), layout(app), "utf8");
  await writeFile(join(base, "lib", "api.ts"), api(app), "utf8");
  process.stdout.write(`${app.dir}\n`);
}
