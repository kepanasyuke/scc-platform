const VIZ_COLORS = {
  red: "#ed131c",
  green: "#16a34a",
  yellow: "#d9a400",
  blue: "#4561c8",
  purple: "#7a4df0",
  track: "#e9e9ec"
};

function vizDonut(value, options = {}) {
  const total = Math.max(0.0001, Number(options.total ?? 100));
  const part = Math.max(0, Math.min(Number(value) || 0, total));
  const share = part / total;
  const size = options.size ?? 96;
  const stroke = options.stroke ?? 9;
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const filled = circumference * share;
  const color = VIZ_COLORS[options.tone ?? "red"] ?? VIZ_COLORS.red;
  const center = options.center ?? `${Math.round(share * 100)}%`;
  const caption = options.caption ?? "";
  return `<div class="viz viz--donut">
    <svg viewBox="0 0 ${size} ${size}" width="${size}" height="${size}" role="img" aria-label="${caption || center}">
      <circle cx="${size / 2}" cy="${size / 2}" r="${radius}" fill="none" stroke="${VIZ_COLORS.track}" stroke-width="${stroke}" />
      <circle cx="${size / 2}" cy="${size / 2}" r="${radius}" fill="none" stroke="${color}" stroke-width="${stroke}"
        stroke-linecap="butt" stroke-dasharray="${filled} ${circumference - filled}"
        transform="rotate(-90 ${size / 2} ${size / 2})" />
      <text x="50%" y="50%" text-anchor="middle" dominant-baseline="central" class="viz__value">${center}</text>
    </svg>
    ${caption ? `<span class="viz__caption">${caption}</span>` : ""}
  </div>`;
}

function vizSparkline(values, options = {}) {
  const points = (values ?? []).map((value) => Number(value) || 0);
  const width = options.width ?? 240;
  const height = options.height ?? 44;
  const color = VIZ_COLORS[options.tone ?? "red"] ?? VIZ_COLORS.red;
  if (points.length < 2) return "";
  const peak = Math.max(...points);
  const floor = Math.min(...points);
  const span = Math.max(1, peak - floor);
  const step = width / (points.length - 1);
  const coordinates = points.map((value, index) => {
    const x = index * step;
    const y = height - ((value - floor) / span) * (height - 6) - 3;
    return [x, y];
  });
  const line = coordinates.map(([x, y], index) => `${index === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`).join(" ");
  const area = `${line} L${width},${height} L0,${height} Z`;
  const last = coordinates[coordinates.length - 1];
  return `<div class="viz viz--spark">
    <svg viewBox="0 0 ${width} ${height}" width="100%" height="${height}" preserveAspectRatio="none" role="img" aria-label="динамика">
      <path d="${area}" fill="${color}" opacity="0.08" />
      <path d="${line}" fill="none" stroke="${color}" stroke-width="1.6" stroke-linejoin="round" stroke-linecap="round" />
      <circle cx="${last[0].toFixed(1)}" cy="${last[1].toFixed(1)}" r="2.4" fill="${color}" />
    </svg>
  </div>`;
}

function vizBars(items, options = {}) {
  const data = (items ?? []).map((item) => ({ label: item.label, value: Number(item.value) || 0 }));
  const peak = Math.max(1, ...data.map((item) => item.value));
  const color = VIZ_COLORS[options.tone ?? "red"] ?? VIZ_COLORS.red;
  return `<div class="viz viz--bars" style="height:${options.height ?? 96}px">
    ${data
      .map(
        (item) => `<div class="viz__bar" title="${item.label}: ${item.value}">
          <span style="height:${Math.max(2, (item.value / peak) * 100)}%;background:${color}"></span>
          <em>${item.label}</em>
        </div>`
      )
      .join("")}
  </div>`;
}

function vizStack(segments, options = {}) {
  const total = Math.max(1, segments.reduce((acc, item) => acc + (Number(item.value) || 0), 0));
  return `<div class="viz viz--stack">
    ${segments
      .map(
        (item) => `<span style="width:${((Number(item.value) || 0) / total) * 100}%;background:${VIZ_COLORS[item.tone ?? "red"]}"
          title="${item.label}: ${item.value}"></span>`
      )
      .join("")}
    <div class="viz__legend">
      ${segments.map((item) => `<i><b style="background:${VIZ_COLORS[item.tone ?? "red"]}"></b>${item.label} ${item.value}</i>`).join("")}
    </div>
  </div>`;
}

function vizBlock(title, body) {
  return `<div class="viz-block">
    <div class="side-block__title">${title}</div>
    ${body}
  </div>`;
}
