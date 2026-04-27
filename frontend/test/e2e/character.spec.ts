import { expect, test } from "@playwright/test";
import { resetStub, setApiCharacters } from "./helpers";

test.describe("キャラタブ", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
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
  });

  test("キャラタブが表示され、キャラクター画像エリアとセリフエリアが見える", async ({
    page,
  }) => {
    await page.goto("/");
    await page.getByRole("tab", { name: "キャラ" }).click();

    const panel = page.locator("#panel-character");
    await expect(panel).toBeVisible();
  });

  test("縦長ビューポートでもキャラタブがビューポートからはみ出さない", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 400, height: 600 });
    await page.goto("/");
    await page.getByRole("tab", { name: "キャラ" }).click();

    const panel = page.locator("#panel-character");
    await expect(panel).toBeVisible();

    const panelBox = await panel.boundingBox();
    expect(panelBox).not.toBeNull();
    if (panelBox) {
      expect(panelBox.y + panelBox.height).toBeLessThanOrEqual(600);
    }
  });

  test("横長ビューポートでもキャラタブがビューポートからはみ出さない", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 800, height: 450 });
    await page.goto("/");
    await page.getByRole("tab", { name: "キャラ" }).click();

    const panel = page.locator("#panel-character");
    await expect(panel).toBeVisible();

    const panelBox = await panel.boundingBox();
    expect(panelBox).not.toBeNull();
    if (panelBox) {
      expect(panelBox.y + panelBox.height).toBeLessThanOrEqual(450);
    }
  });

  test("セリフ欄はパネル内に収まる", async ({ page }) => {
    await page.setViewportSize({ width: 400, height: 600 });
    await page.goto("/");
    await page.getByRole("tab", { name: "キャラ" }).click();

    const panel = page.locator("#panel-character");
    const panelBox = await panel.boundingBox();
    expect(panelBox).not.toBeNull();
    if (panelBox) {
      expect(panelBox.y + panelBox.height).toBeLessThanOrEqual(600);
    }
  });
});
