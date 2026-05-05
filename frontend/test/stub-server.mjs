// Playwright E2E 用スタブサーバー。
// バックエンド (internal/infra/http_stream_player.go) の /events, /api/status,
// /api/characters, /api/history, /api/play, /clips/<id>.wav を Node 標準ライブラリだけでエミュレートする。
//
// テスト側からの制御チャネルとして以下を追加で提供する:
// - POST /__stub/clip             { id?, url?, text, speakerName?, styleName?, timestamp? } → SSE clip 配信
// - POST /__stub/error            { id?, category, message, ... } → SSE error 配信
// - POST /__stub/disconnect       → 全 SSE 接続を一斉切断（再接続テスト用）
// - POST /__stub/api-status       { silent, silentReason, speakers } → /api/status の応答を切替
// - POST /__stub/api-characters   { enabled, characters } → /api/characters の応答を切替
// - POST /__stub/api-history      { entries: [...] } → /api/history の応答を設定
// - POST /__stub/reset            → 内部状態を初期化
//
// VOX_STUB_PORT で待受ポートを変えられる（既定: 8080）。

import http from "node:http";

const PORT = Number(process.env.VOX_STUB_PORT ?? 8080);

// 短い無音 WAV（44 byte ヘッダのみ、PCM mono 8000Hz 8bit、データ長 0）。
// /clips/*.wav が返す。テストは再生成功までは検証しない。
const SILENT_WAV = (() => {
  const buf = Buffer.alloc(44);
  buf.write("RIFF", 0);
  buf.writeUInt32LE(36, 4);
  buf.write("WAVE", 8);
  buf.write("fmt ", 12);
  buf.writeUInt32LE(16, 16);
  buf.writeUInt16LE(1, 20);
  buf.writeUInt16LE(1, 22);
  buf.writeUInt32LE(8000, 24);
  buf.writeUInt32LE(8000, 28);
  buf.writeUInt16LE(1, 32);
  buf.writeUInt16LE(8, 34);
  buf.write("data", 36);
  buf.writeUInt32LE(0, 40);
  return buf;
})();

const DEFAULT_API_STATUS = {
  silent: false,
  silentReason: "",
  speakers: [
    { id: 1, speakerName: "四国めたん", styleName: "ノーマル" },
    { id: 3, speakerName: "ずんだもん", styleName: "ノーマル" },
  ],
};

const DEFAULT_API_CHARACTERS = {
  enabled: false,
  characters: [],
};

const DEFAULT_API_HISTORY = { entries: [] };

let apiStatus = structuredClone(DEFAULT_API_STATUS);
let apiCharacters = structuredClone(DEFAULT_API_CHARACTERS);
let apiHistory = structuredClone(DEFAULT_API_HISTORY);
let nextErrorId = 0;
const subscribers = new Set();

function readJSON(req) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    req.on("data", (c) => chunks.push(c));
    req.on("end", () => {
      const raw = Buffer.concat(chunks).toString("utf8");
      if (raw === "") return resolve({});
      try {
        resolve(JSON.parse(raw));
      } catch (err) {
        reject(err);
      }
    });
    req.on("error", reject);
  });
}

function writeJSON(res, status, body) {
  const payload = Buffer.from(JSON.stringify(body), "utf8");
  res.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Content-Length": payload.length,
    "Cache-Control": "no-store",
  });
  res.end(payload);
}

function broadcast(event, data) {
  const frame = `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
  for (const res of subscribers) {
    res.write(frame);
  }
}

function handleEvents(req, res) {
  res.writeHead(200, {
    "Content-Type": "text/event-stream",
    "Cache-Control": "no-cache",
    Connection: "keep-alive",
  });
  res.write(": ok\n\n");
  subscribers.add(res);
  req.on("close", () => {
    subscribers.delete(res);
  });
}

function handleApiStatus(_req, res) {
  writeJSON(res, 200, apiStatus);
}

function handleApiCharacters(_req, res) {
  writeJSON(res, 200, apiCharacters);
}

function handleApiHistory(_req, res) {
  writeJSON(res, 200, apiHistory);
}

async function handleApiPlay(req, res) {
  const body = await readJSON(req);
  const clips = Array.isArray(body.clips) ? body.clips : [];
  const playbackId = `stub-${Date.now()}`;
  writeJSON(res, 200, {
    playback_id: playbackId,
    clip_count: clips.length,
    silent: false,
  });
}

function handleClipFile(_req, res) {
  res.writeHead(200, {
    "Content-Type": "audio/wav",
    "Content-Length": SILENT_WAV.length,
  });
  res.end(SILENT_WAV);
}

async function handleStubClip(req, res) {
  const body = await readJSON(req);
  const timestamp = typeof body.timestamp === "number" ? body.timestamp : Date.now();
  const payload = {
    url: typeof body.url === "string" ? body.url : `/clips/${timestamp}.wav`,
    text: typeof body.text === "string" ? body.text : "",
    speakerName:
      typeof body.speakerName === "string" ? body.speakerName : "ずんだもん",
    styleName: typeof body.styleName === "string" ? body.styleName : "ノーマル",
    timestamp,
    speakerId: typeof body.speakerId === "number" ? body.speakerId : 0,
  };
  if (typeof body.speed === "number") payload.speed = body.speed;
  if (typeof body.pitch === "number") payload.pitch = body.pitch;
  if (typeof body.intonation === "number") payload.intonation = body.intonation;
  broadcast("clip", payload);
  writeJSON(res, 200, { ok: true, payload });
}

async function handleStubError(req, res) {
  const body = await readJSON(req);
  nextErrorId += 1;
  const payload = {
    id: typeof body.id === "number" ? body.id : nextErrorId,
    category: typeof body.category === "string" ? body.category : "synthesis",
    message: typeof body.message === "string" ? body.message : "stub error",
    timestamp: typeof body.timestamp === "number" ? body.timestamp : Date.now(),
  };
  if (typeof body.path === "string") payload.path = body.path;
  if (typeof body.text === "string") payload.text = body.text;
  if (typeof body.speakerName === "string")
    payload.speakerName = body.speakerName;
  if (typeof body.styleName === "string") payload.styleName = body.styleName;
  broadcast("error", payload);
  writeJSON(res, 200, { ok: true, payload });
}

function handleStubDisconnect(_req, res) {
  for (const sub of subscribers) {
    sub.end();
  }
  subscribers.clear();
  writeJSON(res, 200, { ok: true });
}

async function handleStubApiStatus(req, res) {
  const body = await readJSON(req);
  apiStatus = {
    silent:
      typeof body.silent === "boolean"
        ? body.silent
        : DEFAULT_API_STATUS.silent,
    silentReason:
      typeof body.silentReason === "string"
        ? body.silentReason
        : DEFAULT_API_STATUS.silentReason,
    speakers: Array.isArray(body.speakers)
      ? body.speakers
      : DEFAULT_API_STATUS.speakers,
  };
  writeJSON(res, 200, { ok: true, apiStatus });
}

async function handleStubApiCharacters(req, res) {
  const body = await readJSON(req);
  apiCharacters = {
    enabled:
      typeof body.enabled === "boolean"
        ? body.enabled
        : DEFAULT_API_CHARACTERS.enabled,
    characters: Array.isArray(body.characters)
      ? body.characters
      : DEFAULT_API_CHARACTERS.characters,
  };
  writeJSON(res, 200, { ok: true, apiCharacters });
}

async function handleStubApiHistory(req, res) {
  const body = await readJSON(req);
  const entries = Array.isArray(body.entries)
    ? body.entries.map((e) => ({
        speakerId: 0,
        ...e,
      }))
    : [];
  apiHistory = { entries };
  writeJSON(res, 200, { ok: true, apiHistory });
}

function handleStubReset(_req, res) {
  apiStatus = structuredClone(DEFAULT_API_STATUS);
  apiCharacters = structuredClone(DEFAULT_API_CHARACTERS);
  apiHistory = structuredClone(DEFAULT_API_HISTORY);
  nextErrorId = 0;
  for (const sub of subscribers) {
    sub.end();
  }
  subscribers.clear();
  writeJSON(res, 200, { ok: true });
}

const server = http.createServer((req, res) => {
  const url = new URL(req.url ?? "/", "http://localhost");
  const path = url.pathname;

  if (req.method === "GET" && path === "/events") return handleEvents(req, res);
  if (req.method === "GET" && path === "/api/status")
    return handleApiStatus(req, res);
  if (req.method === "GET" && path === "/api/characters")
    return handleApiCharacters(req, res);
  if (req.method === "GET" && path === "/api/history")
    return handleApiHistory(req, res);
  if (req.method === "POST" && path === "/api/play")
    return handleApiPlay(req, res);
  if (
    req.method === "GET" &&
    path.startsWith("/clips/") &&
    path.endsWith(".wav")
  ) {
    return handleClipFile(req, res);
  }
  if (req.method === "POST" && path === "/__stub/clip")
    return handleStubClip(req, res);
  if (req.method === "POST" && path === "/__stub/error")
    return handleStubError(req, res);
  if (req.method === "POST" && path === "/__stub/disconnect") {
    return handleStubDisconnect(req, res);
  }
  if (req.method === "POST" && path === "/__stub/api-status") {
    return handleStubApiStatus(req, res);
  }
  if (req.method === "POST" && path === "/__stub/api-characters") {
    return handleStubApiCharacters(req, res);
  }
  if (req.method === "POST" && path === "/__stub/api-history") {
    return handleStubApiHistory(req, res);
  }
  if (req.method === "POST" && path === "/__stub/reset")
    return handleStubReset(req, res);

  res.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" });
  res.end("not found");
});

server.listen(PORT, "127.0.0.1", () => {
  console.log(`stub-server listening on http://127.0.0.1:${PORT}`);
});

const shutdown = () => {
  for (const sub of subscribers) sub.end();
  subscribers.clear();
  server.close(() => process.exit(0));
};
process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
