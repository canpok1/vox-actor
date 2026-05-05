import { expect, test } from "@playwright/test";
import { pushClip, resetStub, setApiHistory } from "./helpers";

test.describe("再生キュー", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("連続して clip を受信した場合 Timeline の表示順は受信順になる", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    const baseTs = 1700003000000;
    for (let i = 0; i < 3; i++) {
      await pushClip(request, {
        text: `順序テスト ${i + 1}`,
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: baseTs + i,
      });
    }

    // 全件が Timeline に表示される
    for (let i = 0; i < 3; i++) {
      await expect(
        page.locator(`[data-clip-timestamp="${baseTs + i}"]`),
      ).toBeVisible();
    }

    // 受信順（timestamp 昇順）に並んでいることを確認
    const items = page.locator("[data-clip-timestamp]");
    const count = await items.count();
    expect(count).toBe(3);

    const timestamps = await Promise.all(
      Array.from({ length: count }, (_, i) =>
        items.nth(i).getAttribute("data-clip-timestamp"),
      ),
    );
    const numericTimestamps = timestamps.map(Number);
    for (let i = 1; i < numericTimestamps.length; i++) {
      expect(numericTimestamps[i]).toBeGreaterThan(numericTimestamps[i - 1]);
    }
  });

  test("muted 状態でも Timeline には clip が積まれる", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    // デフォルトは消音チェック ON（audio.muted=true）
    await expect(page.locator("audio")).toHaveJSProperty("muted", true);

    const ts = 1700003100001;
    await pushClip(request, {
      text: "消音中テキスト",
      speakerName: "ずんだもん",
      styleName: "ノーマル",
      timestamp: ts,
    });

    // audio.muted=true でも Timeline には表示される
    const item = page.locator(`[data-clip-timestamp="${ts}"]`);
    await expect(item).toBeVisible();
    await expect(item).toContainText("消音中テキスト");
  });

  test("URL が空の clip は Timeline には表示されるが audio.src は更新されない", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    const ts = 1700003200001;
    await pushClip(request, {
      url: "",
      text: "URL空クリップ",
      speakerName: "ずんだもん",
      styleName: "ノーマル",
      timestamp: ts,
    });

    // Timeline には表示される
    const item = page.locator(`[data-clip-timestamp="${ts}"]`);
    await expect(item).toBeVisible();
    await expect(item).toContainText("URL空クリップ");

    // audio.src は空のまま（再生キューに積まれていない）
    const audioSrc = await page.evaluate(
      () => document.querySelector("audio")?.src ?? "",
    );
    expect(audioSrc).toBe("");
  });

  test("audio.src に clip の URL が反映される（再生キューへの積み込み確認）", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    // 消音を解除して再生キューが有効になる状態にする
    await page.getByRole("checkbox", { name: "消音" }).uncheck();

    const ts = 1700003300001;
    const clipUrl = `/clips/${ts}.wav`;

    const requestPromise = page.waitForRequest(
      (req) => req.url().includes(`/clips/${ts}.wav`),
    );

    await pushClip(request, {
      url: clipUrl,
      text: "再生キューテスト",
      speakerName: "ずんだもん",
      styleName: "ノーマル",
      timestamp: ts,
    });

    await requestPromise;

    // 先読みにより fetch が完了し audio.src に blob URL が設定されている
    const audioSrc = await page.locator("audio").evaluate(
      (el: HTMLAudioElement) => el.src,
    );
    expect(audioSrc).toMatch(/^blob:/);
  });

  test("テスト再生で共有 audio の src が blob URL になる（口パク対象の audioRef を経由）", async ({
    page,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    await page.getByRole("tab", { name: "音声テスト" }).click();
    await page.getByLabel("話者", { exact: true }).selectOption("3");

    const requestPromise = page.waitForRequest(
      (req) => req.url().includes("/api/preview-clip") && req.method() === "POST",
    );

    await page.getByRole("button", { name: /テスト再生/ }).click();
    await requestPromise;

    // 共有 audio 要素の src が blob URL になっている = 口パク対象の audioRef を経由している
    const audioSrc = await page.locator("audio").evaluate(
      (el: HTMLAudioElement) => el.src,
    );
    expect(audioSrc).toMatch(/^blob:/);
  });

  test("履歴「再生」ボタンで共有 audio の src が blob URL になる（口パク対象の audioRef を経由）", async ({
    page,
    request,
  }) => {
    const ts = 1700003400001;
    await setApiHistory(request, [
      {
        text: "履歴再生テスト",
        speakerName: "ずんだもん",
        styleName: "ノーマル",
        timestamp: ts,
      },
    ]);

    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();
    await expect(page.locator(`[data-clip-timestamp="${ts}"]`)).toBeVisible();

    const requestPromise = page.waitForRequest(
      (req) => req.url().includes("/api/preview-clip") && req.method() === "POST",
    );

    await page.locator(`[data-clip-timestamp="${ts}"]`).getByRole("button", { name: "再生" }).click();
    await requestPromise;

    // 共有 audio 要素の src が blob URL になっている
    const audioSrc = await page.locator("audio").evaluate(
      (el: HTMLAudioElement) => el.src,
    );
    expect(audioSrc).toMatch(/^blob:/);
  });

  test("テストタブ表示中でも clip が届いたら audio.src に blob URL が設定される（常時アクティブキュー）", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    // テストタブへ切り替え
    await page.getByRole("tab", { name: "音声テスト" }).click();

    const ts = 1700003500001;
    const clipUrl = `/clips/${ts}.wav`;

    const requestPromise = page.waitForRequest(
      (req) => req.url().includes(`/clips/${ts}.wav`),
    );

    await pushClip(request, {
      url: clipUrl,
      text: "タブ切替後キューテスト",
      speakerName: "ずんだもん",
      styleName: "ノーマル",
      timestamp: ts,
    });

    // テストタブ表示中でもキューが動作し fetch が発生する
    await requestPromise;

    const audioSrc = await page.locator("audio").evaluate(
      (el: HTMLAudioElement) => el.src,
    );
    expect(audioSrc).toMatch(/^blob:/);
  });
});
