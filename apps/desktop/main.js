const { app, BrowserWindow, ipcMain, shell } = require("electron");
const { spawn } = require("node:child_process");
const { join } = require("node:path");
const { existsSync } = require("node:fs");
const { writeFile } = require("node:fs/promises");

const ROOT = join(__dirname, "..", "..");

const services = [
  { id: "cdr-ingest", dir: "01-cdr-ingest", port: 8101, env: "SCC_CDR_INGEST_PORT" },
  { id: "call-analytics", dir: "02-call-analytics", port: 8102, env: "SCC_CALL_ANALYTICS_PORT" },
  { id: "sip-probe", dir: "03-sip-probe", port: 8103, env: "SCC_SIP_PROBE_PORT" },
  { id: "queue-acd", dir: "04-queue-acd", port: 8104, env: "SCC_QUEUE_ACD_PORT" },
  { id: "supervisor-alerts", dir: "05-supervisor-alerts", port: 8105, env: "SCC_SUPERVISOR_ALERTS_PORT" },
  { id: "freeswitch-esl", dir: "06-freeswitch-esl", port: 8106, env: "SCC_FREESWITCH_ESL_PORT" },
  { id: "desk-phone", dir: "07-desk-phone-provisioning", port: 8107, env: "SCC_DESK_PHONE_PORT" },
  { id: "pg-doctor", dir: "08-pg-doctor", port: 8108, env: "SCC_PG_DOCTOR_PORT" }
];

const children = new Map();
let window = null;

function createWindow() {
  const iconPath = join(__dirname, "build", "icon.png");
  window = new BrowserWindow({
    width: 1460,
    height: 920,
    minWidth: 1120,
    minHeight: 700,
    show: false,
    frame: false,
    backgroundColor: "#1c1c1c",
    title: "SCC Platform",
    icon: existsSync(iconPath) ? iconPath : undefined,
    webPreferences: {
      preload: join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false
    }
  });

  window.loadFile(join(__dirname, "renderer", "index.html"));
  window.once("ready-to-show", () => window.show());
  window.webContents.once("did-finish-load", () => {
    window.show();
    window.focus();
  });
  window.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: "deny" };
  });

  if (process.argv.includes("--smoke")) {
    window.webContents.once("did-finish-load", () => {
      setTimeout(async () => {
        const probe = await window.webContents.executeJavaScript(`({
          kpis: document.querySelectorAll(".kpi").length,
          nav: document.querySelectorAll(".nav__item").length,
          rows: document.querySelectorAll("tbody tr").length,
          title: document.getElementById("title").textContent,
          mode: document.getElementById("mode-pill").textContent,
          label: document.getElementById("live-label").textContent
        })`);
        process.stdout.write(`desktop-smoke ${JSON.stringify(probe)}\n`);
        app.quit();
      }, 4000);
    });
  }
}

function startService(service) {
  if (children.has(service.id)) return service;
  const cwd = join(ROOT, "projects", service.dir, "backend");
  if (!existsSync(cwd)) return service;
  const child = spawn("go", ["run", "./cmd/server"], {
    cwd,
    env: { ...process.env, [service.env]: String(service.port), GOWORK: "off" },
    shell: true,
    windowsHide: true
  });
  child.stdout.on("data", (chunk) => process.stdout.write(`[${service.id}] ${chunk}`));
  child.stderr.on("data", (chunk) => process.stderr.write(`[${service.id}] ${chunk}`));
  child.on("exit", () => children.delete(service.id));
  children.set(service.id, child);
  return service;
}

function stopService(id) {
  const child = children.get(id);
  if (child) {
    child.kill();
    children.delete(id);
  }
  return id;
}

app.whenReady().then(() => {
  createWindow();
  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on("window-all-closed", () => {
  for (const id of children.keys()) stopService(id);
  if (process.platform !== "darwin") app.quit();
});

ipcMain.handle("window:minimize", () => window?.minimize());
ipcMain.handle("window:toggle", () => {
  if (!window) return false;
  if (window.isMaximized()) window.unmaximize();
  else window.maximize();
  return window.isMaximized();
});
ipcMain.handle("window:close", () => window?.close());

ipcMain.handle("export:csv", async (_event, name, content) => {
  const target = join(app.getPath("downloads"), name);
  await writeFile(target, content, "utf8");
  return target;
});

ipcMain.handle("services:list", () => services.map((service) => ({ ...service, running: children.has(service.id) })));

ipcMain.handle("config:ports", () => {
  const raw = process.env.SCC_API_PORTS ?? "";
  const overrides = {};
  for (const pair of raw.split(",")) {
    const [id, port] = pair.split("=");
    if (id && port) overrides[id.trim()] = Number(port);
  }
  return overrides;
});
ipcMain.handle("services:start", (_event, id) => {
  const service = services.find((item) => item.id === id);
  return service ? { ...startService(service), running: true } : null;
});
ipcMain.handle("services:start-all", () => {
  services.forEach(startService);
  return services.length;
});
ipcMain.handle("services:stop", (_event, id) => ({ id: stopService(id), running: false }));
