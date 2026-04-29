import { expect, test } from "@playwright/test";
import { resetStub, setApiHistory } from "./helpers";

test.describe("配信タブ: 再生履歴", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("ページ読み込み時に /api/history の履歴がタイムラインに表示される", async ({
    page,
    request,
  }) => {
    await setApiHistory(request, [
      {
        id: 1001,
        text: "履歴テキスト1",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: Date.now(),
      },
      {
        id: 1002,
        text: "履歴テキスト2",
        speakerName: "四国めたん",
        styleName: "ノーマル",
        timestamp: Date.now(),
      },
    ]);

    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    await expect(page.locator('[data-clip-id="1001"]')).toBeVisible();
    await expect(
      page.locator('[data-clip-id="1001"]'),
    ).toContainText("履歴テキスト1");
    await expect(page.locator('[data-clip-id="1002"]')).toBeVisible();
    await expect(
      page.locator('[data-clip-id="1002"]'),
    ).toContainText("履歴テキスト2");
  });

  test("履歴項目の表示後に audio の src が空のまま（音声再生が起きない）", async ({
    page,
    request,
  }) => {
    await setApiHistory(request, [
      {
        id: 2001,
        text: "音声なし履歴",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: Date.now(),
      },
    ]);

    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();
    await expect(page.locator('[data-clip-id="2001"]')).toBeVisible();

    // 履歴ロード後も audio.src が空 = 再生キューに積まれていない。
    const audioSrc = await page.evaluate(
      () => document.querySelector("audio")?.src ?? "",
    );
    expect(audioSrc).toBe("");
  });

  test("viewer 起動直後に履歴リストが末尾（最新エントリ）にスクロールされる", async ({
    page,
    request,
  }) => {
    const entries = Array.from({ length: 20 }, (_, i) => ({
      id: 6000 + i,
      text: `スクロールテスト履歴${i + 1}`,
      speakerName: "ずんだもん",
      styleName: "ノーマル",
      timestamp: Date.now() + i * 1000,
    }));
    await setApiHistory(request, entries);

    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    const lastEntry = page.locator(`[data-clip-id="${entries[entries.length - 1].id}"]`);
    await expect(lastEntry).toBeVisible();
    await expect(lastEntry).toBeInViewport();

    const firstEntry = page.locator(`[data-clip-id="${entries[0].id}"]`);
    await expect(firstEntry).not.toBeInViewport();
  });

  test("履歴が空のとき従来どおりリストは空のまま", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    const list = page.getByRole("list");
    await expect(list).toBeEmpty();
  });

  test("リロード後も最新の /api/history が表示される", async ({
    page,
    request,
  }) => {
    await setApiHistory(request, [
      {
        id: 3001,
        text: "リロード前テキスト",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: Date.now(),
      },
    ]);

    await page.goto("/");
    await expect(page.locator('[data-clip-id="3001"]')).toBeVisible();

    // リロード前後で新しいエントリを追加する。
    await setApiHistory(request, [
      {
        id: 3001,
        text: "リロード前テキスト",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: Date.now(),
      },
      {
        id: 3002,
        text: "リロード後追加テキスト",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: Date.now(),
      },
    ]);

    await page.reload();
    await expect(page.getByText("● 接続中")).toBeVisible();

    await expect(page.locator('[data-clip-id="3002"]')).toBeVisible();
    await expect(
      page.locator('[data-clip-id="3002"]'),
    ).toContainText("リロード後追加テキスト");
  });
});
