const state = {
  active: "cdr-ingest",
  live: false,
  timer: null
};

const PORTS = {
  "cdr-ingest": 8101,
  "call-analytics": 8102,
  "sip-probe": 8103,
  "queue-acd": 8104,
  "supervisor-alerts": 8105,
  "freeswitch-esl": 8106,
  "desk-phone": 8107,
  "pg-doctor": 8108
};

const fmt = (value, digits = 0) =>
  Number(value ?? 0).toLocaleString("ru-RU", { minimumFractionDigits: digits, maximumFractionDigits: digits });

const dur = (seconds) => {
  const total = Math.max(0, Math.round(Number(seconds) || 0));
  return `${String(Math.floor(total / 60)).padStart(2, "0")}:${String(total % 60).padStart(2, "0")}`;
};

const tag = (text, tone = "") => `<span class="tag ${tone ? `tag--${tone}` : ""}">${text}</span>`;

async function getJson(port, path) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 1500);
  try {
    const response = await fetch(`http://127.0.0.1:${port}${path}`, { signal: controller.signal });
    return response.ok ? await response.json() : null;
  } catch {
    return null;
  } finally {
    clearTimeout(timer);
  }
}

const kpi = (label, value, hint = "", tone = "") => ({ label, value, hint, tone });

const SERVICES = [
  {
    id: "cdr-ingest",
    num: "01",
    title: "CDR Ingest",
    subtitle: "Приём и запись детализации вызовов: батчи до 5 000 записей, дедупликация по call_id, партиции по суткам.",
    port: 8101,
    async load() {
      const stats = await getJson(PORTS["cdr-ingest"], "/v1/cdr/stats");
      const recent = await getJson(PORTS["cdr-ingest"], "/v1/cdr/recent?limit=10");
      const data = stats ?? { received: 18420, accepted: 18364, rejected: 12, duplicates: 44, answer_rate: 93.4, asa_seconds: 9.2, sla_percent: 96.1, batches: 612 };
      const records = recent?.records ?? [
        { call_id: "9f1c…a12", direction: "inbound", queue: "admission", trunk: "mti-sip-01", agent_ext: "101", ring_seconds: 7, talk_seconds: 214, sla_met: true },
        { call_id: "3ab7…c04", direction: "inbound", queue: "support", trunk: "mti-sip-02", agent_ext: "118", ring_seconds: 26, talk_seconds: 96, sla_met: false },
        { call_id: "77d0…f91", direction: "outbound", queue: "callback", trunk: "mti-out-01", agent_ext: "104", ring_seconds: 4, talk_seconds: 61, sla_met: true }
      ];
      return {
        live: Boolean(stats),
        kpis: [
          kpi("Принято записей", fmt(data.accepted), `батчей: ${fmt(data.batches)}`),
          kpi("Дубликаты", fmt(data.duplicates), "ON CONFLICT", ""),
          kpi("Отклонено", fmt(data.rejected), "валидация схемы", data.rejected > 0 ? "warning" : "success"),
          kpi("Доля отвеченных", `${fmt(data.answer_rate, 1)}%`, "за сутки", "success"),
          kpi("ASA", dur(data.asa_seconds), "до ответа", ""),
          kpi("SLA", `${fmt(data.sla_percent, 1)}%`, "порог 20 с", data.sla_percent > 95 ? "success" : "danger")
        ],
        table: {
          title: "Последние вызовы",
          hint: "telephony.cdr",
          columns: ["Call ID", "Направление", "Очередь", "Транк", "Агент", "Ожидание", "Разговор", "SLA"],
          rows: records.map((item) => [
            `<span class="mono">${item.call_id}</span>`,
            tag(item.direction, item.direction === "inbound" ? "info" : ""),
            item.queue,
            item.trunk,
            item.agent_ext,
            dur(item.ring_seconds),
            dur(item.talk_seconds),
            tag(item.sla_met ? "в норме" : "нарушен", item.sla_met ? "ok" : "bad")
          ])
        },
        side: [
          { title: "Конвейер", rows: [["Приём", "HTTP батч"], ["Валидация", "до записи"], ["Запись", "plpgsql upsert"], ["Партиция", "сутки"]] },
          { title: "Индексы", rows: [["BRIN", "started_at"], ["Partial", "sla_met = false"], ["GIN", "to_tsvector"]] }
        ]
      };
    }
  },
  {
    id: "call-analytics",
    num: "02",
    title: "Call Analytics",
    subtitle: "Нагрузка по часам, ASA, p95 и SLA по очередям на continuous aggregates TimescaleDB.",
    port: 8102,
    async load() {
      const summary = await getJson(PORTS["call-analytics"], "/v1/analytics/summary");
      const queues = await getJson(PORTS["call-analytics"], "/v1/analytics/queues");
      const hourly = await getJson(PORTS["call-analytics"], "/v1/analytics/hourly");
      const data = summary ?? { total: 1600, answered: 1493, sla: 94.3, asa: 8.7, p95_wait: 27.4, peak_hour: "12:00", peak_count: 214 };
      const list = queues?.queues ?? [
        { queue: "admission", total: 412, answered: 389, asa: 7.2, p95_wait: 22.1, sla: 96.4, abandonment: 5.6 },
        { queue: "support", total: 508, answered: 471, asa: 9.8, p95_wait: 31.2, sla: 91.2, abandonment: 7.3 },
        { queue: "exams", total: 396, answered: 371, asa: 6.4, p95_wait: 18.4, sla: 97.8, abandonment: 6.3 }
      ];
      const buckets = hourly?.buckets ?? [
        { hour: "09:00", total: 142 }, { hour: "10:00", total: 178 }, { hour: "11:00", total: 196 },
        { hour: "12:00", total: 214 }, { hour: "13:00", total: 188 }, { hour: "14:00", total: 164 }, { hour: "15:00", total: 152 }
      ];
      return {
        live: Boolean(summary),
        kpis: [
          kpi("Всего вызовов", fmt(data.total), "за сутки"),
          kpi("Отвечено", fmt(data.answered), `${fmt((data.answered / Math.max(1, data.total)) * 100, 1)}%`, "success"),
          kpi("SLA", `${fmt(data.sla, 1)}%`, "порог 20 с", data.sla >= 90 ? "success" : "danger"),
          kpi("ASA", dur(data.asa), "среднее ожидание"),
          kpi("p95 ожидания", dur(data.p95_wait), "хвост распределения", "warning"),
          kpi("Пиковый час", data.peak_hour, `${fmt(data.peak_count)} вызовов`)
        ],
        table: {
          title: "Очереди",
          hint: "queue_kpi",
          columns: ["Очередь", "Вызовы", "Отвечено", "ASA", "p95", "SLA", "Отказы"],
          rows: list.map((item) => [
            `<strong>${item.queue}</strong>`,
            fmt(item.total),
            fmt(item.answered),
            dur(item.asa),
            dur(item.p95_wait),
            tag(`${fmt(item.sla, 1)}%`, item.sla >= 95 ? "ok" : item.sla >= 90 ? "warn" : "bad"),
            `${fmt(item.abandonment, 1)}%`
          ])
        },
        bars: buckets.map((bucket) => ({ label: bucket.hour.slice(0, 2), value: bucket.total })),
        side: [
          { title: "Расчёт", rows: [["ASA", "avg(wait)"], ["p95", "percentile_cont"], ["SLA", "wait ≤ 20 c"], ["Отказы", "answered_at is null"]] },
          { title: "TimescaleDB", rows: [["hourly_calls", "caggr 5 мин"], ["retention", "400 дней"], ["партиции", "1 сутки"]] }
        ]
      };
    }
  },
  {
    id: "sip-probe",
    num: "03",
    title: "SIP Probe",
    subtitle: "Мониторинг SIP-транков: собственный OPTIONS-клиент по UDP/TLS, разбор статус-линии и заголовков, RTT.",
    port: 8103,
    async load() {
      const summary = await getJson(PORTS["sip-probe"], "/v1/trunks/summary");
      const probes = await getJson(PORTS["sip-probe"], "/v1/trunks/last");
      const trunks = await getJson(PORTS["sip-probe"], "/v1/trunks");
      const data = summary ?? { trunks: 5, healthy: 3, failed: 2 };
      const list = probes?.probes ?? [
        { trunk: "mti-sip-01", code: 200, reason: "OK", rtt_ms: 12.4, healthy: true },
        { trunk: "mti-sip-02", code: 200, reason: "OK", rtt_ms: 18.9, healthy: true },
        { trunk: "mti-sip-03", code: 403, reason: "Forbidden", rtt_ms: 44.1, healthy: false },
        { trunk: "mti-webrtc-01", code: 0, reason: "", rtt_ms: 0, healthy: false, error: "i/o timeout" }
      ];
      const capacity = (trunks?.trunks ?? []).reduce((acc, item) => ({ load: acc.load + item.channels, total: acc.total + item.capacity }), { load: 247, total: 590 });
      return {
        live: Boolean(summary),
        kpis: [
          kpi("Транков", fmt(data.trunks), "SIP + WebRTC"),
          kpi("Доступно", fmt(data.healthy), "OPTIONS 2xx", "success"),
          kpi("Аварий", fmt(data.failed), "код вне 2xx", data.failed > 0 ? "danger" : "success"),
          kpi("Загрузка линий", `${fmt((capacity.load / Math.max(1, capacity.total)) * 100, 1)}%`, `${capacity.load} / ${capacity.total} каналов`)
        ],
        table: {
          title: "Последние пробы",
          hint: "sip.options",
          columns: ["Транк", "Код", "Reason", "RTT, мс", "Статус"],
          rows: list.map((item) => [
            `<strong>${item.trunk}</strong>`,
            item.code === 0 ? tag("нет ответа", "bad") : tag(String(item.code), item.healthy ? "ok" : "bad"),
            item.reason || item.error || "—",
            fmt(item.rtt_ms, 1),
            tag(item.healthy ? "healthy" : "failed", item.healthy ? "ok" : "bad")
          ])
        },
        side: [
          { title: "Проба", rows: [["Метод", "OPTIONS"], ["Транспорт", "UDP / TLS"], ["Таймаут", "3 с"], ["RTT", "от отправки"]] },
          { title: "FreeSWITCH", rows: [["sofia", "external"], ["profile", "mti-sip"], ["регистрация", "300 с"]] }
        ]
      };
    }
  },
  {
    id: "queue-acd",
    num: "04",
    title: "Queue ACD",
    subtitle: "Распределение вызовов: приоритеты, навыки, выбор агента и SLA-таймеры. События уходят в шину.",
    port: 8104,
    async load() {
      const queues = await getJson(PORTS["queue-acd"], "/v1/queues");
      const agents = await getJson(PORTS["queue-acd"], "/v1/agents");
      const list = queues?.queues ?? [
        { queue: "admission", waiting: 21, ready: 2, busy: 4, oldest_wait: 128, avg_wait: 61.4, sla_percent: 88.2 },
        { queue: "support", waiting: 9, ready: 1, busy: 3, oldest_wait: 41, avg_wait: 22.7, sla_percent: 96.4 }
      ];
      const staff = agents?.agents ?? [
        { ext: "101", name: "Анна Кузнецова", team: "приёмная комиссия", state: "ready", handled: 34 },
        { ext: "107", name: "Мария Соколова", team: "поддержка", state: "busy", handled: 28 }
      ];
      const waiting = list.reduce((acc, item) => acc + item.waiting, 0);
      const ready = staff.filter((agent) => agent.state === "ready").length;
      return {
        live: Boolean(queues),
        kpis: [
          kpi("В ожидании", fmt(waiting), "все очереди", waiting > 15 ? "danger" : "success"),
          kpi("Готовы", fmt(ready), "агентов на линии", "success"),
          kpi("Заняты", fmt(staff.filter((agent) => agent.state === "busy").length), "разговаривают", "warning"),
          kpi("Очередей", fmt(list.length), "активных")
        ],
        table: {
          title: "Очереди",
          hint: "acd.snapshot",
          columns: ["Очередь", "Ожидание", "Готовы", "Заняты", "Старейшее", "SLA"],
          rows: list.map((item) => [
            `<strong>${item.queue}</strong>`,
            fmt(item.waiting),
            fmt(item.ready),
            fmt(item.busy),
            dur(item.oldest_wait),
            tag(`${fmt(item.sla_percent, 1)}%`, item.sla_percent >= 95 ? "ok" : "warn")
          ])
        },
        side: [
          { title: "Правила", rows: [["Приоритет", "VIP → FIFO"], ["Выбор агента", "меньше вызовов"], ["Навыки", "skill match"], ["Таймаут", "20 с"]] },
          { title: "Шина", rows: [["enqueue", "calls.queue.enqueue"], ["assign", "calls.queue.assign"]] }
        ]
      };
    }
  },
  {
    id: "supervisor-alerts",
    num: "05",
    title: "Supervisor Alerts",
    subtitle: "Движок правил для супервизоров: пороги, уровни критичности и окна тишины против шторма событий.",
    port: 8105,
    async load() {
      const stats = await getJson(PORTS["supervisor-alerts"], "/v1/stats");
      const alerts = await getJson(PORTS["supervisor-alerts"], "/v1/alerts?open=true");
      const rules = await getJson(PORTS["supervisor-alerts"], "/v1/rules");
      const data = stats ?? { open: 3, critical: 2, rules: 7 };
      const list = alerts?.alerts ?? [
        { id: "a-0001", severity: "critical", rule: "долгое ожидание", subject: "admission", message: "порог 90 с" },
        { id: "a-0002", severity: "warning", rule: "нет свободных агентов", subject: "admission", message: "порог 1" },
        { id: "a-0003", severity: "critical", rule: "транк недоступен", subject: "outbound", message: "порог 1" }
      ];
      return {
        live: Boolean(stats),
        kpis: [
          kpi("Открытые", fmt(data.open), "не подтверждены", "warning"),
          kpi("Критичные", fmt(data.critical), "требуют реакции", data.critical > 0 ? "danger" : "success"),
          kpi("Правил", fmt(data.rules), "в движке", "info"),
          kpi("Окно тишины", "120–600 с", "анти-шторм")
        ],
        table: {
          title: "Активные оповещения",
          hint: "telephony.alerts",
          columns: ["ID", "Уровень", "Правило", "Объект", "Сообщение"],
          rows: list.map((item) => [
            `<span class="mono">${item.id}</span>`,
            tag(item.severity, item.severity === "critical" ? "bad" : "warn"),
            `<strong>${item.rule}</strong>`,
            item.subject,
            item.message
          ])
        },
        side: [
          { title: "Правила", rows: (rules?.rules ?? []).slice(0, 5).map((rule) => [rule.name, String(rule.threshold)]) },
          { title: "Доставка", rows: [["Шина", "alerts.raised"], ["Пульт", "push"], ["Дублирование", "Telegram"]] }
        ]
      };
    }
  },
  {
    id: "freeswitch-esl",
    num: "06",
    title: "FreeSWITCH ESL",
    subtitle: "Консоль управления каналами: ESL-клиент, состояние cs_*, кодеки и оценка MOS по RTP-метрикам.",
    port: 8106,
    async load() {
      const status = await getJson(PORTS["freeswitch-esl"], "/v1/esl/status");
      const summary = await getJson(PORTS["freeswitch-esl"], "/v1/channels/summary");
      const channels = await getJson(PORTS["freeswitch-esl"], "/v1/channels");
      const data = summary ?? { total: 4, active: 4, talking: 3, avg_mos: 4.21 };
      const list = channels?.channels ?? [
        { uuid: "b1f2a3c4-0001", name: "sofia/external/79001234567", destination: "1001", state: "cs_execute", codec: "PCMU", mos: 4.31 },
        { uuid: "b1f2a3c4-0002", name: "sofia/internal/104", destination: "queue_admission", state: "answered", codec: "OPUS", mos: 4.52 },
        { uuid: "b1f2a3c4-0003", name: "sofia/external/79161234567", destination: "queue_support", state: "cs_ring", codec: "PCMU", mos: 3.88 }
      ];
      return {
        live: Boolean(summary),
        kpis: [
          kpi("Активные каналы", fmt(data.active), `${status?.version ?? "FreeSWITCH 1.10"}`, "info"),
          kpi("Разговаривают", fmt(data.talking), "cs_execute / answered", "success"),
          kpi("Средний MOS", fmt(data.avg_mos, 2), "по RTP", data.avg_mos >= 4 ? "success" : "warning"),
          kpi("ESL порт", "8021", status?.connected === false ? "offline" : "connected", status?.connected === false ? "danger" : "success")
        ],
        table: {
          title: "Каналы",
          hint: "show channels",
          columns: ["UUID", "Канал", "Назначение", "Состояние", "Кодек", "MOS"],
          rows: list.map((item) => [
            `<span class="mono">${item.uuid}</span>`,
            item.name,
            item.destination,
            tag(item.state, item.state === "cs_ring" ? "warn" : "ok"),
            item.codec,
            fmt(item.mos, 2)
          ])
        },
        side: [
          { title: "ESL", rows: [["auth", "ClueCon"], ["api", "status"], ["bgapi", "originate"], ["reconnect", "backoff"]] },
          { title: "Качество", rows: [["MOS 4.0+", "норма"], ["джиттер", "≤ 30 мс"], ["потери", "≤ 1%"]] }
        ]
      };
    }
  },
  {
    id: "desk-phone",
    num: "07",
    title: "Desk Phone",
    subtitle: "Провижининг стационарных аппаратов: MAC, привязка номера и XML-конфигурация для Yealink и Grandstream.",
    port: 8107,
    async load() {
      const devices = await getJson(PORTS["desk-phone"], "/v1/devices");
      const list = devices?.devices ?? [
        { mac: "805e0c1a2b3c", vendor: "yealink", model: "t33g", extension: "101", location: "приёмная, корпус А", provisioned: true },
        { mac: "000b8212aa01", vendor: "grandstream", model: "grp2602", extension: "107", location: "поддержка", provisioned: true },
        { mac: "000b8212aa02", vendor: "grandstream", model: "gxp1625", extension: "118", location: "экзаменационный центр", provisioned: false }
      ];
      const done = list.filter((item) => item.provisioned).length;
      return {
        live: Boolean(devices),
        kpis: [
          kpi("Аппаратов", fmt(list.length), "в парке"),
          kpi("Настроены", fmt(done), "получили конфиг", "success"),
          kpi("Ожидают", fmt(list.length - done), "первое включение", "warning"),
          kpi("Шаблонов", "4", "Yealink · Grandstream")
        ],
        table: {
          title: "Реестр устройств",
          hint: "provision.devices",
          columns: ["MAC", "Модель", "Номер", "Размещение", "Статус"],
          rows: list.map((item) => [
            `<span class="mono">${item.mac}</span>`,
            `${item.vendor} ${item.model}`,
            `<strong>${item.extension}</strong>`,
            item.location,
            tag(item.provisioned ? "настроен" : "ожидает", item.provisioned ? "ok" : "warn")
          ])
        },
        side: [
          { title: "Выдача", rows: [["Запрос", "GET /v1/config/{mac}.xml"], ["Формат", "XML"], ["MAC", "12 hex"]] },
          { title: "Шаблоны", rows: [["Yealink", "T33G · T46U"], ["Grandstream", "GRP2602 · GXP1625"]] }
        ]
      };
    }
  },
  {
    id: "pg-doctor",
    num: "08",
    title: "PG Doctor",
    subtitle: "Диагностика PostgreSQL: доля seq scan, дорогие запросы, рекомендации индексов и разбор дедлоков.",
    port: 8108,
    async load() {
      const report = await getJson(PORTS["pg-doctor"], "/v1/report");
      const data = report ?? {
        health: "critical",
        samples: 5,
        slow_queries: 3,
        recommendations: [
          { table: "cdr", severity: "critical", problem: "seq scan 98%", suggestion: "CREATE INDEX CONCURRENTLY idx_cdr_queue ON cdr (queue)", estimated_gain: "−78%" },
          { table: "query_samples", severity: "critical", problem: "seq scan 99%", suggestion: "CREATE INDEX ON query_samples (fingerprint)", estimated_gain: "−79%" }
        ],
        deadlocks: [{ at: "09:14:02", victim: "process 18422", relation: "telephony.agents", advice: "сначала agents, затем queues" }]
      };
      return {
        live: Boolean(report),
        kpis: [
          kpi("Состояние", data.health, "снимок pg_stat_statements", data.health === "critical" ? "danger" : "success"),
          kpi("Запросов", fmt(data.samples), "в снимке"),
          kpi("Медленных", fmt(data.slow_queries), "среднее > 200 мс", "warning"),
          kpi("Дедлоков", fmt((data.deadlocks ?? []).length), "в логе", "danger")
        ],
        table: {
          title: "Рекомендации",
          hint: "ops.query_samples",
          columns: ["Таблица", "Уровень", "Проблема", "Что сделать", "Эффект"],
          rows: (data.recommendations ?? []).map((item) => [
            `<strong>${item.table}</strong>`,
            tag(item.severity, item.severity === "critical" ? "bad" : "warn"),
            item.problem,
            `<span class="mono">${item.suggestion}</span>`,
            item.estimated_gain
          ])
        },
        side: [
          { title: "Дедлоки", rows: (data.deadlocks ?? []).map((item) => [item.victim, item.relation]) },
          { title: "Порядок блокировок", rows: [["1", "agents"], ["2", "queues"], ["3", "cdr"]] }
        ]
      };
    }
  }
];

function renderNav(activeId) {
  const nav = document.getElementById("nav");
  nav.innerHTML = SERVICES.map(
    (service) => `
      <button class="nav__item ${service.id === activeId ? "nav__item--active" : ""}" data-id="${service.id}">
        <span class="nav__dot" data-dot="${service.id}"></span>
        <span>${service.title}</span>
        <span class="nav__num">${service.num}</span>
      </button>`
  ).join("");
  nav.querySelectorAll(".nav__item").forEach((button) => {
    button.addEventListener("click", () => select(button.dataset.id));
  });
}

function renderKpis(items) {
  document.getElementById("kpis").innerHTML = items
    .map(
      (item) => `
      <article class="kpi ${item.tone ? `kpi--${item.tone}` : ""}">
        <span class="kpi__label">${item.label}</span>
        <span class="kpi__value">${item.value}</span>
        ${item.hint ? `<span class="kpi__hint">${item.hint}</span>` : ""}
      </article>`
    )
    .join("");
}

function renderTable(table) {
  document.getElementById("table-title").textContent = table.title;
  document.getElementById("table-hint").textContent = table.hint ?? "";
  const rows = table.rows.length
    ? table.rows
        .map((row) => `<tr>${row.map((cell) => `<td>${cell}</td>`).join("")}</tr>`)
        .join("")
    : `<tr><td colspan="${table.columns.length}"><div class="empty">Нет данных за выбранный период</div></td></tr>`;
  document.getElementById("table").innerHTML = `
    <table>
      <thead><tr>${table.columns.map((column) => `<th>${column}</th>`).join("")}</tr></thead>
      <tbody>${rows}</tbody>
    </table>`;
}

function renderSide(blocks, bars) {
  const charts = bars?.length
    ? `<div class="side-block">
         <div class="side-block__title">Нагрузка по часам</div>
         <div class="bars">
           ${bars
             .map((bar) => {
               const peak = Math.max(...bars.map((item) => item.value), 1);
               return `<div class="bars__item"><div class="bars__bar" style="height:${(bar.value / peak) * 100}%"></div><span class="bars__label">${bar.label}</span></div>`;
             })
             .join("")}
         </div>
       </div>`
    : "";
  const rows = (blocks ?? [])
    .map(
      (block) => `
      <div class="side-block">
        <div class="side-block__title">${block.title}</div>
        ${block.rows.map((row) => `<div class="side-block__row"><span>${row[0]}</span><span>${row[1]}</span></div>`).join("")}
      </div>`
    )
    .join("");
  document.getElementById("side").innerHTML = charts + rows;
}

function setMode(live, service) {
  const pill = document.getElementById("mode-pill");
  pill.textContent = live ? "live" : "демо";
  pill.className = `pill ${live ? "pill--ok" : "pill--warn"}`;
  document.getElementById("port-pill").textContent = `:${service.port}`;
  const dot = document.getElementById("live-dot");
  dot.className = `dot ${live ? "dot--ok" : "dot--off"}`;
  document.getElementById("live-label").textContent = live
    ? `${service.title} · API online`
    : `${service.title} · демо-данные`;
  const navDot = document.querySelector(`[data-dot="${service.id}"]`);
  if (navDot) navDot.className = `nav__dot ${live ? "nav__dot--ok" : ""}`;
}

async function select(id) {
  state.active = id;
  const service = SERVICES.find((item) => item.id === id) ?? SERVICES[0];
  renderNav(id);
  document.getElementById("title").textContent = service.title;
  document.getElementById("subtitle").textContent = service.subtitle;
  document.getElementById("eyebrow").textContent = `SCC Platform · ${service.num}`;
  const view = await service.load();
  state.live = view.live;
  setMode(view.live, service);
  renderKpis(view.kpis);
  renderTable(view.table);
  renderSide(view.side, view.bars);
  clearInterval(state.timer);
  state.timer = setInterval(() => refresh(true), 6000);
}

async function refresh(silent) {
  const service = SERVICES.find((item) => item.id === state.active);
  if (!service) return;
  const view = await service.load();
  state.live = view.live;
  setMode(view.live, service);
  renderKpis(view.kpis);
  renderTable(view.table);
  renderSide(view.side, view.bars);
  if (!silent) document.getElementById("refresh").blur();
}

function tick() {
  document.getElementById("clock").textContent = new Date().toLocaleTimeString("ru-RU");
}

document.getElementById("refresh").addEventListener("click", () => refresh(false));
document.querySelectorAll(".titlebar__controls button").forEach((button) => {
  button.addEventListener("click", () => {
    const action = button.dataset.action;
    if (action === "minimize") window.scc?.window.minimize();
    if (action === "toggle") window.scc?.window.toggle();
    if (action === "close") window.scc?.window.close();
  });
});
document.getElementById("start-all").addEventListener("click", async () => {
  await window.scc?.services.startAll();
  setTimeout(() => refresh(true), 2000);
});
document.getElementById("stop-all").addEventListener("click", async () => {
  const list = await window.scc?.services.list();
  for (const service of list ?? []) await window.scc?.services.stop(service.id);
  setTimeout(() => refresh(true), 800);
});

tick();
setInterval(tick, 1000);
async function boot() {
  const overrides = await window.scc?.ports?.();
  if (overrides) Object.assign(PORTS, overrides);
  await select(state.active);
}

boot();
