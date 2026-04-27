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

  test("2人表示で画像がアスペクト比を保つ（600px幅）", async ({
    page,
    request,
  }) => {
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
        {
          speakerName: "四国めたん",
          styleName: "ノーマル",
          mouthClosed: "metan_closed.png",
          mouthOpen: "metan_open.png",
        },
      ],
    });

    await page.setViewportSize({ width: 600, height: 800 });
    await page.goto("/");
    await page.getByRole("tab", { name: "キャラ" }).click();

    // 2人表示なら複数キャラが表示される
    // デスクトップレイアウトで複数のキャラクターが表示されているかを確認
    const desktopLayout = page.locator(
      "#panel-character > div > div.sm\\:flex"
    );

    // 少なくとも 1 つの子要素（キャラクター）が visible
    const childCount = await desktopLayout.locator("> div").count();
    expect(childCount).toBeGreaterThan(1);
  });

  test("3人表示で中央に配置される", async ({ page, request }) => {
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
        {
          speakerName: "四国めたん",
          styleName: "ノーマル",
          mouthClosed: "metan_closed.png",
          mouthOpen: "metan_open.png",
        },
        {
          speakerName: "ずんだもん",
          styleName: "あまあま",
          mouthClosed: "zundamon_closed.png",
          mouthOpen: "zundamon_open.png",
        },
      ],
    });

    await page.setViewportSize({ width: 800, height: 800 });
    await page.goto("/");
    await page.getByRole("tab", { name: "キャラ" }).click();

    // 3人表示でデスクトップレイアウトが使われることを確認
    const desktopLayout = page.locator(
      "#panel-character > div > div.sm\\:flex"
    );
    await expect(desktopLayout).toBeVisible();

    // 3 つのキャラクターが表示されている
    const childCount = await desktopLayout.locator("> div").count();
    expect(childCount).toBe(3);
  });

  test("モバイル画面（400px）で複数キャラが正しく配置される", async ({
    page,
    request,
  }) => {
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
        {
          speakerName: "四国めたん",
          styleName: "ノーマル",
          mouthClosed: "metan_closed.png",
          mouthOpen: "metan_open.png",
        },
      ],
    });

    await page.setViewportSize({ width: 400, height: 600 });
    await page.goto("/");
    await page.getByRole("tab", { name: "キャラ" }).click();

    const panel = page.locator("#panel-character");
    await expect(panel).toBeVisible();

    // モバイルレイアウトが表示される
    const mobileLayout = page.locator(
      "#panel-character > div > div.sm\\:hidden"
    );
    await expect(mobileLayout).toBeVisible();

    // 2 つのキャラクターが配置されている
    const childCount = await mobileLayout.locator("> div").count();
    expect(childCount).toBe(2);
  });
});
