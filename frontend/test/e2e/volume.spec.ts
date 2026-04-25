import { expect, test } from "@playwright/test";
import { readLocalStorage, resetStub } from "./helpers";

test.describe("音量・消音コントロール", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("音量スライダーの値が localStorage に保存され、リロード後も復元される", async ({
    page,
  }) => {
    await page.goto("/");
    const slider = page.getByRole("slider", { name: "音量" });
    await slider.fill("23");

    await expect
      .poll(() => readLocalStorage(page, "vox-actor.stream.volume"))
      .toBe("23");

    await page.reload();
    await expect(page.getByRole("slider", { name: "音量" })).toHaveValue("23");
  });

  test("消音チェックボックスの操作で audio.muted が連動する", async ({
    page,
  }) => {
    await page.goto("/");

    await expect(page.locator("audio")).toHaveJSProperty("muted", true);

    await page.getByRole("checkbox", { name: "消音" }).uncheck();
    await expect(page.locator("audio")).toHaveJSProperty("muted", false);

    await page.getByRole("checkbox", { name: "消音" }).check();
    await expect(page.locator("audio")).toHaveJSProperty("muted", true);
  });
});
