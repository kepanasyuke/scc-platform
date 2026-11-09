function illustrationEmpty(kind = "table") {
  const captions = {
    table: "Нет данных за выбранный период",
    calls: "Вызовов пока не было",
    search: "Ничего не нашлось"
  };
  return `<div class="illustration">
    <svg viewBox="0 0 120 84" width="120" height="84" fill="none" aria-hidden="true">
      <rect x="14" y="10" width="92" height="58" rx="6" stroke="#d9d9de" stroke-width="1.4" />
      <path d="M14 24h92" stroke="#d9d9de" stroke-width="1.4" />
      <circle cx="22" cy="17" r="1.6" fill="#c9c9d0" />
      <circle cx="28" cy="17" r="1.6" fill="#c9c9d0" />
      <circle cx="34" cy="17" r="1.6" fill="#c9c9d0" />
      <path d="M26 38h30M26 46h44M26 54h22" stroke="#e2e2e7" stroke-width="1.6" stroke-linecap="round" />
      <rect x="66" y="34" width="28" height="26" rx="4" stroke="#ed131c" stroke-opacity="0.35" stroke-width="1.4" stroke-dasharray="3 3" />
      <path d="M80 42v8M80 54h.01" stroke="#ed131c" stroke-opacity="0.55" stroke-width="1.6" stroke-linecap="round" />
    </svg>
    <span>${captions[kind] ?? captions.table}</span>
  </div>`;
}

function illustrationOffline() {
  return `<svg viewBox="0 0 64 64" width="40" height="40" fill="none" aria-hidden="true">
    <rect x="8" y="12" width="48" height="34" rx="5" stroke="#c9a227" stroke-width="1.8" />
    <path d="M8 22h48" stroke="#c9a227" stroke-width="1.8" />
    <circle cx="14" cy="17" r="1.6" fill="#c9a227" />
    <circle cx="19" cy="17" r="1.6" fill="#c9a227" />
    <path d="M24 34h16M32 30v8" stroke="#c9a227" stroke-width="1.8" stroke-linecap="round" />
    <path d="M20 52h24" stroke="#c9a227" stroke-width="1.8" stroke-linecap="round" />
  </svg>`;
}

function illustrationLoading() {
  return `<div class="illustration illustration--muted">
    <svg viewBox="0 0 120 84" width="120" height="84" fill="none" aria-hidden="true">
      <rect x="14" y="10" width="92" height="58" rx="6" stroke="#e4e4e7" stroke-width="1.4" />
      <path d="M14 24h92" stroke="#e4e4e7" stroke-width="1.4" />
      <path d="M26 38h44M26 46h56M26 54h32" stroke="#efeff1" stroke-width="1.6" stroke-linecap="round" />
    </svg>
    <span>Загружаю данные…</span>
  </div>`;
}
