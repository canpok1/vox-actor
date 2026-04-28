import { expect, test } from "@playwright/test";
import { readLocalStorage, resetStub } from "./helpers";

test.describe("各種設定の localStorage 永続化", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("履歴上限・表示トグルが localStorage に保存され、リロード後も復元される", async ({
    page,
  }) => {
    await page.goto("/");

    await page.getByLabel("履歴上限").selectOption("50");
    await page.getByRole("checkbox", { name: "キャラ名" }).uncheck();
    await page.getByRole("checkbox", { name: "スタイル" }).uncheck();
    await page.getByRole("checkbox", { name: "時刻" }).uncheck();

    await expect
      .poll(() => readLocalStorage(page, "vox-actor.stream.historySize"))
      .toBe("50");
    await expect
      .poll(() => readLocalStorage(page, "vox-actor.stream.showSpeakerName"))
      .toBe("false");
    await expect
      .poll(() => readLocalStorage(page, "vox-actor.stream.showStyleName"))
      .toBe("false");
    await expect
      .poll(() => readLocalStorage(page, "vox-actor.stream.showTimestamp"))
      .toBe("false");

    await page.reload();
    await expect(page.getByLabel("履歴上限")).toHaveValue("50");
    await expect(
      page.getByRole("checkbox", { name: "キャラ名" }),
    ).not.toBeChecked();
    await expect(
      page.getByRole("checkbox", { name: "スタイル" }),
    ).not.toBeChecked();
    await expect(
      page.getByRole("checkbox", { name: "時刻" }),
    ).not.toBeChecked();
  });
});
