import { expect, test } from "@playwright/test";
import { readLocalStorage, resetStub } from "./helpers";

test.describe("タブ切替", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("配信タブと音声テストタブを切り替えられる", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    const streamPanel = page.locator("#panel-stream");
    const testPanel = page.locator("#panel-test");

    await expect(streamPanel).toBeVisible();
    await expect(testPanel).toBeHidden();

    await page.getByRole("tab", { name: "音声テスト" }).click();
    await expect(testPanel).toBeVisible();
    await expect(streamPanel).toBeHidden();

    await page.getByRole("tab", { name: "配信" }).click();
    await expect(streamPanel).toBeVisible();
    await expect(testPanel).toBeHidden();
  });

  test("選択したタブはリロード後も復元される", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("tab", { name: "音声テスト" }).click();

    await expect
      .poll(() => readLocalStorage(page, "vox-actor.stream.activeTab"))
      .toBe("test");

    await page.reload();
    await expect(page.locator("#panel-test")).toBeVisible();
    await expect(page.locator("#panel-stream")).toBeHidden();
  });
});
