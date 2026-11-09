const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("scc", {
  ports: () => ipcRenderer.invoke("config:ports"),
  export: { csv: (name, content) => ipcRenderer.invoke("export:csv", name, content) },
  window: {
    minimize: () => ipcRenderer.invoke("window:minimize"),
    toggle: () => ipcRenderer.invoke("window:toggle"),
    close: () => ipcRenderer.invoke("window:close")
  },
  services: {
    list: () => ipcRenderer.invoke("services:list"),
    start: (id) => ipcRenderer.invoke("services:start", id),
    startAll: () => ipcRenderer.invoke("services:start-all"),
    stop: (id) => ipcRenderer.invoke("services:stop", id)
  }
});
