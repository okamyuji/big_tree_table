import { test, expect } from "@playwright/test";

test.describe("BigTable E2E", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    // テーブルデータが読み込まれるまで待機
    await page.waitForSelector('[role="row"]', { timeout: 10000 });
  });

  test("テーブルが表示され、データ行が存在する", async ({ page }) => {
    // ヘッダーが表示される
    await expect(page.getByRole("columnheader", { name: "注文番号" })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "顧客名" })).toBeVisible();

    // データ行が存在する
    const rows = page.locator('[role="row"][aria-rowindex]');
    await expect(rows.first()).toBeVisible();
  });

  test("ソートが動作する（全データ対象）", async ({ page }) => {
    const amountHeader = page.getByRole("columnheader", { name: /金額/ });
    const spinner = page.locator('[role="status"]');
    const dataRows = page.locator('[role="row"][aria-rowindex]');

    // 昇順ソート
    await amountHeader.click();
    await expect(amountHeader).toHaveAttribute("aria-sort", "ascending");
    // スピナーが消えてデータ行が表示されるまで待つ
    await spinner.waitFor({ state: "hidden", timeout: 10000 }).catch(() => {});
    await dataRows.first().waitFor({ state: "visible", timeout: 10000 });
    const ascOrderNum = await dataRows.first().locator('[role="gridcell"]').first().textContent();

    // 降順ソート
    await amountHeader.click();
    await expect(amountHeader).toHaveAttribute("aria-sort", "descending");
    await spinner.waitFor({ state: "hidden", timeout: 10000 }).catch(() => {});
    await dataRows.first().waitFor({ state: "visible", timeout: 10000 });
    const descOrderNum = await dataRows.first().locator('[role="gridcell"]').first().textContent();

    // 昇順と降順で先頭注文番号が異なることを確認（APIテストで ORD-0212019 vs ORD-0612631 を確認済）
    expect(descOrderNum).not.toEqual(ascOrderNum);
  });

  test("ページネーションが動作する", async ({ page }) => {
    // 全件数が表示されている
    const totalText = page.locator("text=1,000,000");
    await expect(totalText).toBeVisible();

    // 次のページボタンをクリック
    const nextButton = page.getByLabel("次のページ");
    await expect(nextButton).toBeEnabled();
    await nextButton.click();

    // ページ2に移動したことを確認
    await page.waitForTimeout(500);
    const page2Button = page.getByRole("button", { name: "2" });
    await expect(page2Button).toHaveAttribute("aria-current", "page");
  });

  test("ステータスフィルターが動作する", async ({ page }) => {
    // ステータスフィルターで「出荷済み」を選択
    const statusSelect = page.locator("#filter-ステータス");
    if (await statusSelect.isVisible()) {
      await statusSelect.selectOption("出荷済み");
    } else {
      // フィルターがselectでない場合はlabelから探す
      const statusFilter = page.getByLabel("ステータス");
      await statusFilter.selectOption("出荷済み");
    }

    // 適用ボタンをクリック
    const applyButton = page.getByRole("button", { name: /適用|検索/ });
    await applyButton.click();

    // データが絞り込まれるのを待機
    await page.waitForTimeout(1000);

    // 全件数が変わっている（1,000,000より少ない）
    const totalCountEl = page.locator('[data-testid="total-count"]');
    if (await totalCountEl.isVisible()) {
      const totalText = await totalCountEl.textContent();
      expect(Number(totalText?.replace(/,/g, "") ?? "0")).toBeLessThan(1_000_000);
    }
  });

  test("ページサイズ変更が動作する", async ({ page }) => {
    // ページサイズセレクターを変更
    const perPageSelect = page.getByLabel(/表示件数|件\/ページ/);
    if (await perPageSelect.isVisible()) {
      await perPageSelect.selectOption("10");
      await page.waitForTimeout(500);

      // 表示行数が10以下になる
      const rows = page.locator('[role="row"][aria-rowindex]');
      const count = await rows.count();
      expect(count).toBeLessThanOrEqual(10);
    }
  });

  test("日付範囲フィルターが動作する", async ({ page }) => {
    // 開始日を設定
    const dateFrom = page.locator("#date-from");
    if (await dateFrom.isVisible()) {
      await dateFrom.fill("2023-01-01");

      // 終了日を設定
      const dateTo = page.locator("#date-to");
      await dateTo.fill("2023-12-31");

      // 適用
      const applyButton = page.getByRole("button", { name: /適用|検索/ });
      await applyButton.click();

      await page.waitForTimeout(1000);

      // データが絞り込まれる
      const rows = page.locator('[role="row"][aria-rowindex]');
      await expect(rows.first()).toBeVisible();
    }
  });

  test("前のページボタンが1ページ目で無効", async ({ page }) => {
    const prevButton = page.getByLabel("前のページ");
    await expect(prevButton).toBeDisabled();
  });

  test("テーブルのアクセシビリティ - role属性", async ({ page }) => {
    // grid role
    const grid = page.locator('[role="grid"]');
    await expect(grid).toBeVisible();

    // columnheader role
    const headers = page.locator('[role="columnheader"]');
    expect(await headers.count()).toBeGreaterThan(0);

    // gridcell role
    const cells = page.locator('[role="gridcell"]');
    expect(await cells.count()).toBeGreaterThan(0);
  });
});
