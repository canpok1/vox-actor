import { expect, test } from "@playwright/test";
import { resetStub, setApiCharacters } from "./helpers";

test.describe("キャラ画像表示（配信タブ統合）", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("「キャラ」タブが存在しないこと", async ({ page }) => {
    await page.goto("/");
    await expect(
      page.getByRole("tab", { name: "キャラ" }),
    ).not.toBeVisible();
  });

  test("charactersEnabled=true のとき「キャラ画像を表示」チェックボックスが配信タブに表示される", async ({
    page,
    request,
  }) => {
    await setApiCharacters(request, {
      enabled: true,
      characters: [
        {
          speakerName: "ずんだもん",
          styleName: "ノーマル",
          mouthClosed: "zundamon_closed.png",
          mouthOpen: "zundamon_open.png",
        },
      ],
    });

    await page.goto("/");
    await expect(
      page.getByRole("checkbox", { name: "キャラ画像を表示" }),
    ).toBeVisible();
  });

  test("charactersEnabled=false のとき「キャラ画像を表示」チェックボックスが非表示になる", async ({
    page,
    request,
  }) => {
    await setApiCharacters(request, { enabled: false, characters: [] });

    await page.goto("/");
    await expect(
      page.getByRole("checkbox", { name: "キャラ画像を表示" }),
    ).not.toBeVisible();
  });
});
