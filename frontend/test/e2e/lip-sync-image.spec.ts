import { expect, test } from "@playwright/test";
import { pushClip, resetStub, setApiCharacters } from "./helpers";

const CHARACTER = {
  speakerName: "ずんだもん",
  styleName: "ノーマル",
  mouthClosed: "zundamon_closed.png",
  mouthOpen: "zundamon_open.png",
};

test.describe("LipSyncImage 表示", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
  });

  test("charactersEnabled=false のとき「キャラ画像」チェックボックスが表示されず画像エリアも非表示", async ({
    page,
    request,
  }) => {
    await setApiCharacters(request, { enabled: false, characters: [] });

    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    await expect(
      page.getByRole("checkbox", { name: "キャラ画像" }),
    ).not.toBeVisible();
    await expect(
      page.locator('img[alt="character mouth closed"]'),
    ).not.toBeVisible();
  });

  test("charactersEnabled=true でキャラ画像チェックON・clip受信後に口パク画像が表示される", async ({
    page,
    request,
  }) => {
    await setApiCharacters(request, {
      enabled: true,
      characters: [CHARACTER],
    });

    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    // 消音解除（再生キューを有効にするため）
    await page.getByRole("checkbox", { name: "消音" }).uncheck();

    // キャラ画像チェックを ON にする
    await page
      .getByRole("checkbox", { name: "キャラ画像" })
      .check();

    const ts = 1700004000001;

    // clip を待ち受けてから受信
    const requestPromise = page.waitForRequest(
      (req) => req.url().includes(`/clips/${ts}.wav`),
    );

    await pushClip(request, {
      url: `/clips/${ts}.wav`,
      text: "口パクテスト",
      speakerName: CHARACTER.speakerName,
      styleName: CHARACTER.styleName,
      timestamp: ts,
    });

    // audio.src が更新されるのを待つ（再生が開始された証拠）
    await requestPromise;

    // LipSyncImage の img 要素が存在する
    await expect(
      page.locator('img[alt="character mouth closed"]'),
    ).toBeAttached();
    await expect(
      page.locator('img[alt="character mouth open"]'),
    ).toBeAttached();
  });

  test("「キャラ画像」チェックを OFF にするとキャラ画像エリアが非表示になる", async ({
    page,
    request,
  }) => {
    await setApiCharacters(request, {
      enabled: true,
      characters: [CHARACTER],
    });

    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    const checkbox = page.getByRole("checkbox", { name: "キャラ画像" });
    await expect(checkbox).toBeVisible();

    // チェックボックスが ON になっていることを確認（デフォルト状態）してから OFF にする
    if (await checkbox.isChecked()) {
      await checkbox.uncheck();
    }

    // 口パク画像エリアが非表示
    await expect(
      page.locator('img[alt="character mouth closed"]'),
    ).not.toBeVisible();
  });

  test("キャラに対応しない speakerName の clip 受信時は画像が表示されない", async ({
    page,
    request,
  }) => {
    await setApiCharacters(request, {
      enabled: true,
      characters: [CHARACTER],
    });

    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    await page.getByRole("checkbox", { name: "キャラ画像" }).check();

    // 登録キャラと一致しない話者の clip を受信
    const ts = 1700004100001;
    await pushClip(request, {
      url: `/clips/${ts}.wav`,
      text: "不一致話者テスト",
      speakerName: "四国めたん",
      styleName: "ノーマル",
      timestamp: ts,
    });

    // タイムラインには表示される
    await expect(page.locator(`[data-clip-timestamp="${ts}"]`)).toBeVisible();

    // キャラ対応なしのためLipSyncImage は表示されない
    await expect(
      page.locator('img[alt="character mouth closed"]'),
    ).not.toBeVisible();
  });
});
