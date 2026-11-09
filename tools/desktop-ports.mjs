import { readFileSync, writeFileSync } from "node:fs";

const file = process.argv[2];
const mapping = {
  8101: 'PORTS["cdr-ingest"]',
  8102: 'PORTS["call-analytics"]',
  8103: 'PORTS["sip-probe"]',
  8104: 'PORTS["queue-acd"]',
  8105: 'PORTS["supervisor-alerts"]',
  8106: 'PORTS["freeswitch-esl"]',
  8107: 'PORTS["desk-phone"]',
  8108: 'PORTS["pg-doctor"]'
};

let text = readFileSync(file, "utf8");
for (const [port, replacement] of Object.entries(mapping)) {
  text = text.replaceAll(`getJson(${port}, `, `getJson(${replacement}, `);
}

const header = `const PORTS = {
  "cdr-ingest": 8101,
  "call-analytics": 8102,
  "sip-probe": 8103,
  "queue-acd": 8104,
  "supervisor-alerts": 8105,
  "freeswitch-esl": 8106,
  "desk-phone": 8107,
  "pg-doctor": 8108
};

`;
text = text.replace("const fmt =", `${header}const fmt =`);
writeFileSync(file, text, "utf8");
process.stdout.write("порты вынесены в PORTS\n");
