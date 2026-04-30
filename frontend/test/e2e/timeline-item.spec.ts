import { expect, test } from "@playwright/test";
import { pushClip, pushError, resetStub } from "./helpers";

test.describe("TimelineItem の個別 UI 要素", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test.describe("clip: メタ情報の表示切替", () => {
    const ts = 1700001000001;

    test("「キャラ名」ON のとき話者名が表示される", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await expect(page.getByText("● 接続中")).toBeVisible();

      await pushClip(request, {
        text: "キャラ名テスト",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: ts,
      });

      const item = page.locator(`[data-clip-timestamp="${ts}"]`);
      await expect(item).toBeVisible();
      // デフォルトは「キャラ名」チェック ON
      await expect(item).toContainText("ずんだもん");
    });

    test("「キャラ名」OFF のとき話者名が非表示になる", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await page.getByRole("checkbox", { name: "キャラ名" }).uncheck();

      await pushClip(request, {
        text: "キャラ名OFFテスト",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: ts,
      });

      const item = page.locator(`[data-clip-timestamp="${ts}"]`);
      await expect(item).toBeVisible();
      await expect(item).not.toContainText("ずんだもん");
    });

    test("「スタイル」ON のときスタイル名が表示される", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await expect(page.getByText("● 接続中")).toBeVisible();

      await pushClip(request, {
        text: "スタイルテスト",
        speakerName: "四国めたん",
        styleName: "ノーマル",
        timestamp: ts,
      });

      const item = page.locator(`[data-clip-timestamp="${ts}"]`);
      await expect(item).toBeVisible();
      await expect(item).toContainText("ノーマル");
    });

    test("「スタイル」OFF のときスタイル名が非表示になる", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await page.getByRole("checkbox", { name: "スタイル" }).uncheck();

      await pushClip(request, {
        text: "スタイルOFFテスト",
        speakerName: "四国めたん",
        styleName: "ノーマル",
        timestamp: ts,
      });

      const item = page.locator(`[data-clip-timestamp="${ts}"]`);
      await expect(item).toBeVisible();
      await expect(item).not.toContainText("ノーマル");
    });

    test("「時刻」ON のときタイムスタンプが表示される", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await page.getByRole("checkbox", { name: "時刻" }).check();

      const fixedTs = 1700000000000; // 2023-11-15 00:00:00 JST
      await pushClip(request, {
        text: "時刻テスト",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: fixedTs,
      });

      const item = page.locator(`[data-clip-timestamp="${fixedTs}"]`);
      await expect(item).toBeVisible();
      // 時刻表示は HH:MM:SS 形式
      const text = await item.textContent();
      expect(text).toMatch(/\d{2}:\d{2}:\d{2}/);
    });

    test("「時刻」OFF のときタイムスタンプが非表示になる", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await page.getByRole("checkbox", { name: "時刻" }).uncheck();

      const fixedTs = 1700000000000;
      await pushClip(request, {
        text: "時刻OFFテスト",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: fixedTs,
      });

      const item = page.locator(`[data-clip-timestamp="${fixedTs}"]`);
      await expect(item).toBeVisible();
      const text = await item.textContent();
      expect(text).not.toMatch(/\d{2}:\d{2}:\d{2}/);
    });
  });

  test.describe("error: エラーカテゴリ別ラベル表示", () => {
    test("synthesis エラーは「合成エラー」ラベルが表示される", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await expect(page.getByText("● 接続中")).toBeVisible();

      await pushError(request, {
        id: 1,
        category: "synthesis",
        message: "合成に失敗しました",
      });

      const item = page.locator('[data-error-id="1"]');
      await expect(item).toBeVisible();
      await expect(item).toContainText("合成エラー");
      await expect(item).toContainText("合成に失敗しました");
    });

    test("file エラーは「ファイルエラー」ラベルが表示される", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await expect(page.getByText("● 接続中")).toBeVisible();

      await pushError(request, {
        id: 2,
        category: "file",
        message: "ファイルが見つかりません",
      });

      const item = page.locator('[data-error-id="2"]');
      await expect(item).toBeVisible();
      await expect(item).toContainText("ファイルエラー");
    });

    test("connection エラーは「接続エラー」ラベルが表示される", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await expect(page.getByText("● 接続中")).toBeVisible();

      await pushError(request, {
        id: 3,
        category: "connection",
        message: "接続が切断されました",
      });

      const item = page.locator('[data-error-id="3"]');
      await expect(item).toBeVisible();
      await expect(item).toContainText("接続エラー");
    });

    test("error に speakerName と text が含まれている場合は表示される", async ({
      page,
      request,
    }) => {
      await page.goto("/");
      await expect(page.getByText("● 接続中")).toBeVisible();

      await pushError(request, {
        id: 4,
        category: "synthesis",
        message: "合成エラー",
        text: "失敗したテキスト",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
      });

      const item = page.locator('[data-error-id="4"]');
      await expect(item).toBeVisible();
      await expect(item).toContainText("失敗したテキスト");
      await expect(item).toContainText("ずんだもん");
    });
  });
});

test.describe("Timeline: 履歴件数上限", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("履歴件数が上限を超えると古いエントリが削除される", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    // 履歴上限を最小値 10 に設定
    await page.getByLabel("履歴上限").selectOption("10");

    const baseTs = 1700002000000;

    // 11 件送信（最初の 1 件が溢れる）。
    // url="" にすることで再生キューに積まれず playingClipTimestamp が null のまま保たれる。
    // これにより trimTimeline の保護条件に引っかからず純粋な件数上限の挙動を検証できる。
    for (let i = 0; i < 11; i++) {
      await pushClip(request, {
        url: "",
        text: `溢れテスト ${i + 1}`,
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: baseTs + i,
      });
    }

    // 最新の 10 件は表示される
    await expect(
      page.locator(`[data-clip-timestamp="${baseTs + 10}"]`),
    ).toBeVisible();
    await expect(
      page.locator(`[data-clip-timestamp="${baseTs + 1}"]`),
    ).toBeVisible();

    // 最古のエントリは削除されている
    await expect(
      page.locator(`[data-clip-timestamp="${baseTs}"]`),
    ).not.toBeVisible();
  });
});
