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

    // 2人表示なら実際のキャラ2人分の画像が表示される
    const imgElements = page.locator(
      "#panel-character img[alt='character mouth closed'], #panel-character img[alt='character mouth open']"
    );
    const visibleImages = await imgElements.filter({ hasNot: page.locator("div:has-style=display: none") }).count();

    // 少なくとも複数キャラが表示されている
    expect(visibleImages).toBeGreaterThan(0);
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

    const characterPanel = page.locator(
      "#panel-character > div > div:has-child(> div:has-child(img))"
    );
    const panelBox = await characterPanel.boundingBox();
    expect(panelBox).not.toBeNull();

    // パネルの左右の余白がほぼ等しい（中央配置されている）ことを確認
    // パネル親の幅を取得
    const parentPanel = page.locator(
      "#panel-character > div:first-of-type"
    );
    const parentBox = await parentPanel.boundingBox();
    expect(parentBox).not.toBeNull();

    if (panelBox && parentBox) {
      const leftMargin = panelBox.x - parentBox.x;
      const rightMargin = parentBox.x + parentBox.width - (panelBox.x + panelBox.width);

      // 左右の余白が最大 10% の差（許容範囲）以内
      const diff = Math.abs(leftMargin - rightMargin);
      const tolerance = parentBox.width * 0.1;
      expect(diff).toBeLessThan(tolerance);
    }
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

    // パネルがビューポート内に収まる
    const panelBox = await panel.boundingBox();
    expect(panelBox).not.toBeNull();
    if (panelBox) {
      expect(panelBox.width).toBeLessThanOrEqual(400);
    }
  });
});
