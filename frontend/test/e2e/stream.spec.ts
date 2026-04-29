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
});
