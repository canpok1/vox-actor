import { expect, test } from "@playwright/test";
import { readLocalStorage, resetStub } from "./helpers";

test.describe("音声テストタブ", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("話者選択は localStorage に保存され、リロード後も復元される", async ({
    page,
  }) => {
    await page.goto("/");
    await page.getByRole("tab", { name: "音声テスト" }).click();

    const selector = page.getByLabel("話者", { exact: true });
    await expect(selector).toBeVisible();
    await selector.selectOption("1");

    await expect
      .poll(() => readLocalStorage(page, "vox-actor.stream.testSpeakerId"))
      .toBe("1");

    await page.reload();
    await page.getByRole("tab", { name: "音声テスト" }).click();
    await expect(page.getByLabel("話者", { exact: true })).toHaveValue("1");
  });

  test("テスト再生ボタンで POST /api/preview-clip に正しいパラメータが送信される", async ({
    page,
  }) => {
    await page.goto("/");
    await page.getByRole("tab", { name: "音声テスト" }).click();
    await page.getByLabel("話者", { exact: true }).selectOption("3");

    const requestPromise = page.waitForRequest(
      (req) => req.url().includes("/api/preview-clip") && req.method() === "POST",
    );

    await page.getByRole("button", { name: /テスト再生/ }).click();
    const req = await requestPromise;
    const body = req.postDataJSON() as {
      text: string;
      speaker_id: number;
    };
    expect(body.speaker_id).toBe(3);
    expect(body.text).toBeTruthy();
  });
});
