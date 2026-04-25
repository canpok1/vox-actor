import { expect, test } from "@playwright/test";
import { readLocalStorage, resetStub } from "./helpers";

test.describe("音声テストタブ", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("話者選択は localStorage に保存され、リロード後も復元される", async ({ page }) => {
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

  test("テスト再生ボタンで /test-clip にリクエストが飛び、audio.src も更新される", async ({
    page,
  }) => {
    await page.goto("/");
    await page.getByRole("tab", { name: "音声テスト" }).click();
    await page.getByLabel("話者", { exact: true }).selectOption("3");

    const requestPromise = page.waitForRequest(
      (req) =>
        req.url().includes("/test-clip") && req.url().includes("speaker=3"),
    );

    await page.getByRole("button", { name: /テスト再生/ }).click();
    const req = await requestPromise;
    expect(req.url()).toContain("speaker=3");

    await expect(page.locator("audio")).toHaveJSProperty(
      "src",
      new URL("/test-clip?speaker=3", page.url()).toString(),
    );
  });
});
