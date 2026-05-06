import { expect, test } from "@playwright/test";
import { disconnectAll, pushClip, pushError, resetStub } from "./helpers";

test.describe("配信タブ: SSE と Timeline", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("SSE 接続後、clip イベントが Timeline に反映される", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    const ts = 1700000000101;
    await pushClip(request, {
      url: `/clips/${ts}.wav`,
      text: "こんにちはなのだ",
      speakerName: "ずんだもん",
      styleName: "ノーマル",
      timestamp: ts,
    });

    const item = page.locator(`[data-clip-timestamp="${ts}"]`);
    await expect(item).toBeVisible();
    await expect(item).toContainText("こんにちはなのだ");
    await expect(item).toContainText("ずんだもん");
  });

  test("error イベントが Timeline にエラーとして表示される", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    await pushError(request, {
      id: 1,
      category: "synthesis",
      message: "合成に失敗しました",
      text: "失敗テキスト",
      speakerName: "四国めたん",
      styleName: "ノーマル",
    });

    const errorItem = page.locator('[data-error-id="1"]');
    await expect(errorItem).toBeVisible();
    await expect(errorItem).toContainText("合成エラー");
    await expect(errorItem).toContainText("合成に失敗しました");
  });

  test("SSE 切断後、自動再接続される", async ({ page, request }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    await disconnectAll(request);
    await expect(page.getByText("● 切断")).toBeVisible();
    // useEventSource の再接続タイマー（2 秒）+ 初回接続待ちぶんを見込んで余裕を持たせる。
    await expect(page.getByText("● 接続中")).toBeVisible({ timeout: 10_000 });
  });

  test("リングバッファ 20 件上限超過時に古いものから破棄される", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    const baseTs = 1700000001000;
    // 21 件の clip イベントを送信する（デフォルト historySize=20 を超える）
    for (let i = 0; i < 21; i++) {
      await pushClip(request, {
        text: `クリップ${i + 1}`,
        timestamp: baseTs + i,
      });
    }

    // 最後の clip（baseTs+20）が表示されるまで待つ（21 件全受信・リングバッファ適用済みを確認）
    const lastItem = page.locator(`[data-clip-timestamp="${baseTs + 20}"]`);
    await expect(lastItem).toBeVisible();

    // 最初の clip（baseTs）はリングバッファで破棄されているはず
    const firstItem = page.locator(`[data-clip-timestamp="${baseTs}"]`);
    await expect(firstItem).not.toBeVisible();
  });

  test("複数ブラウザへの同時 SSE 配信", async ({ browser, request }) => {
    const page1 = await browser.newPage();
    const page2 = await browser.newPage();

    try {
      await page1.goto("/");
      await page2.goto("/");
      await expect(page1.getByText("● 接続中")).toBeVisible();
      await expect(page2.getByText("● 接続中")).toBeVisible();

      const ts = 1700000002001;
      await pushClip(request, {
        text: "マルチブラウザテスト",
        timestamp: ts,
      });

      // 両ページに clip が表示される
      const item1 = page1.locator(`[data-clip-timestamp="${ts}"]`);
      await expect(item1).toBeVisible();
      await expect(item1).toContainText("マルチブラウザテスト");

      const item2 = page2.locator(`[data-clip-timestamp="${ts}"]`);
      await expect(item2).toBeVisible();
      await expect(item2).toContainText("マルチブラウザテスト");
    } finally {
      await page1.close();
      await page2.close();
    }
  });
});
