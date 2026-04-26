import { type APIRequestContext, expect, type Page } from "@playwright/test";

// stub-server.mjs の制御用エンドポイントは Vite dev server (`/`) からは proxy されない。
// テストでは APIRequestContext を使って stub の素のポートに直接叩く。
// VOX_STUB_PORT を変えた場合は同じ値を渡すこと（playwright.config.ts と揃える）。
const STUB_PORT = process.env.VOX_STUB_PORT ?? "8080";
const STUB_BASE_URL =
  process.env.VOX_STUB_BASE_URL ?? `http://127.0.0.1:${STUB_PORT}`;

export interface ClipPayload {
  id?: number;
  url?: string;
  text: string;
  speakerName?: string;
  styleName?: string;
  timestamp?: number;
}

export interface ErrorPayload {
  id?: number;
  category: "synthesis" | "file" | "connection";
  message: string;
  timestamp?: number;
  path?: string;
  text?: string;
  speakerName?: string;
  styleName?: string;
}

export interface ApiStatusOverride {
  silent?: boolean;
  silentReason?: string;
  speakers?: Array<{ id: number; speakerName: string; styleName: string }>;
}

export interface ApiCharactersOverride {
  enabled?: boolean;
  characters?: Array<{
    speakerName: string;
    styleName: string;
    mouthClosed: string;
    mouthOpen: string;
  }>;
}

export async function resetStub(request: APIRequestContext): Promise<void> {
  const resp = await request.post(`${STUB_BASE_URL}/__stub/reset`);
  expect(resp.ok()).toBeTruthy();
}

export async function setApiStatus(
  request: APIRequestContext,
  override: ApiStatusOverride,
): Promise<void> {
  const resp = await request.post(`${STUB_BASE_URL}/__stub/api-status`, {
    data: override,
  });
  expect(resp.ok()).toBeTruthy();
}

export async function setApiCharacters(
  request: APIRequestContext,
  override: ApiCharactersOverride,
): Promise<void> {
  const resp = await request.post(`${STUB_BASE_URL}/__stub/api-characters`, {
    data: override,
  });
  expect(resp.ok()).toBeTruthy();
}

export async function pushClip(
  request: APIRequestContext,
  payload: ClipPayload,
): Promise<void> {
  const resp = await request.post(`${STUB_BASE_URL}/__stub/clip`, {
    data: payload,
  });
  expect(resp.ok()).toBeTruthy();
}

export async function pushError(
  request: APIRequestContext,
  payload: ErrorPayload,
): Promise<void> {
  const resp = await request.post(`${STUB_BASE_URL}/__stub/error`, {
    data: payload,
  });
  expect(resp.ok()).toBeTruthy();
}

export async function disconnectAll(request: APIRequestContext): Promise<void> {
  const resp = await request.post(`${STUB_BASE_URL}/__stub/disconnect`);
  expect(resp.ok()).toBeTruthy();
}

// localStorage を読み取るユーティリティ。永続化テストで利用する。
export async function readLocalStorage(
  page: Page,
  key: string,
): Promise<string | null> {
  return page.evaluate((k) => window.localStorage.getItem(k), key);
}
