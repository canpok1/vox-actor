import { expect, test } from "@playwright/test";
import { pushClip, resetStub, setApiStatus } from "./helpers";

test.describe("無音モード", () => {
  test.beforeEach(async ({ request }) => {
    await resetStub(request);
    await setApiStatus(request, {
      silent: true,
      silentReason: "VOICEVOX が起動していません",
      speakers: [],
    });
  });

  test("SilentBadge が表示され、音声テストタブは描画されない", async ({
    page,
  }) => {
    await page.goto("/");
    await expect(
      page.getByRole("button", { name: /無音モード/ }),
    ).toBeVisible();
    await expect(page.getByRole("tab", { name: "配信" })).toBeVisible();
    await expect(page.getByRole("tab", { name: "音声テスト" })).toHaveCount(0);
  });

  test("URL 空の clip イベントもタイムラインに表示される", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    await expect(page.getByText("● 接続中")).toBeVisible();

    const ts = 1700000000501;
    await pushClip(request, {
      url: "",
      text: "無音モードで配信されたテキスト",
      speakerName: "ずんだもん",
      styleName: "ノーマル",
      timestamp: ts,
    });

    const item = page.locator(`[data-clip-timestamp="${ts}"]`);
    await expect(item).toBeVisible();
    await expect(item).toContainText("無音モードで配信されたテキスト");
  });
});
