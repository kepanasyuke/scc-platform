const ICON_PATHS = {
  database: [
    "M12 3c-4.97 0-9 1.34-9 3s4.03 3 9 3 9-1.34 9-3-4.03-3-9-3Z",
    "M21 12c0 1.66-4.03 3-9 3s-9-1.34-9-3",
    "M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"
  ],
  chart: ["M3 3v18h18", "M7 16v-4", "M12 16V8", "M17 16v-7"],
  pulse: ["M22 12h-4l-3 9L9 3l-3 9H2"],
  users: [
    "M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2",
    "M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z",
    "M22 21v-2a4 4 0 0 0-3-3.87",
    "M16 3.13a4 4 0 0 1 0 7.75"
  ],
  bell: ["M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9", "M10.3 21a1.94 1.94 0 0 0 3.4 0"],
  phone: [
    "M22 16.92v3a2 2 0 0 1-2.18 2 19.8 19.8 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6A19.8 19.8 0 0 1 2.12 4.18 2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.9.34 1.85.57 2.81.7A2 2 0 0 1 22 16.92Z"
  ],
  monitor: ["M2 3h20v14H2z", "M8 21h8", "M12 17v4"],
  gauge: ["M12 14l4-4", "M3.5 18a9.5 9.5 0 1 1 17 0", "M12 18h.01"],
  search: ["M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16Z", "M21 21l-4.35-4.35"],
  refresh: ["M20.5 5.5v5h-5", "M20.5 10.5a8.5 8.5 0 1 0-2.2 8.6"],
  download: ["M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4", "M7 10l5 5 5-5", "M12 15V3"],
  alert: ["M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z", "M12 9v4", "M12 17h.01"],
  play: ["M6 4l14 8-14 8V4Z"],
  minimize: ["M5 12h14"],
  maximize: ["M4 4h16v16H4z"],
  close: ["M18 6 6 18", "M6 6l12 12"],
  stop: ["M6 6h12v12H6z"],
  refreshAll: ["M3 12a9 9 0 0 1 15.5-6.2L21 8", "M21 3v5h-5", "M21 12a9 9 0 0 1-15.5 6.2L3 16", "M3 21v-5h5"]
};

function icon(name, size = 16) {
  const paths = ICON_PATHS[name] ?? ICON_PATHS.gauge;
  const body = paths.map((path) => `<path d="${path}"/>`).join("");
  return `<svg class="icon" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none"
    stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"
    aria-hidden="true" focusable="false">${body}</svg>`;
}

const SERVICE_ICONS = {
  "cdr-ingest": "database",
  "call-analytics": "chart",
  "sip-probe": "pulse",
  "queue-acd": "users",
  "supervisor-alerts": "bell",
  "freeswitch-esl": "phone",
  "desk-phone": "monitor",
  "pg-doctor": "gauge"
};
