import { expect, test, type Page, type TestInfo } from "@playwright/test";

const TREE_API_PATTERN = /\/api\/order-tree/;
const MAX_INITIAL_TREE_API_MS = 3_000;
const MAX_INITIAL_RENDER_MS = 5_000;

async function waitForTreeTable(page: Page) {
  const startedAt = Date.now();
  const responsePromise = page.waitForResponse((response) => TREE_API_PATTERN.test(response.url()));

  await page.goto("/");

  const response = await responsePromise;
  const apiMs = Date.now() - startedAt;
  expect(response.ok()).toBeTruthy();

  await expect(page.getByRole("heading", { name: "受注管理ツリーテーブル" })).toBeVisible();
  await expect(page.getByRole("columnheader", { name: "階層 / 注文番号" })).toBeVisible();
  await expect(page.locator('[role="row"][aria-rowindex]').first()).toBeVisible();

  return {
    renderMs: Date.now() - startedAt,
    apiMs,
  };
}

async function expectVisibleRowsAreNotBlank(page: Page) {
  const rows = page.locator('[role="row"][aria-rowindex]');
  const rowCount = await rows.count();
  expect(rowCount).toBeGreaterThan(0);

  const firstColumnTexts = await rows
    .locator('[role="gridcell"]')
    .evaluateAll((cells) =>
      cells.filter((_, index) => index % 10 === 0).map((cell) => cell.textContent?.trim() ?? ""),
    );

  expect(firstColumnTexts.length).toBeGreaterThan(0);
  expect(firstColumnTexts.every((text) => text.length > 0)).toBeTruthy();
}

async function scrollGridAndCollectRows(page: Page) {
  return page.locator('[role="grid"]').evaluate(async (grid) => {
    const results = [];

    for (const ratio of [0, 0.25, 0.5, 0.75, 1]) {
      grid.scrollTop = (grid.scrollHeight - grid.clientHeight) * ratio;
      await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));

      const rows = Array.from(document.querySelectorAll('[role="row"][aria-rowindex]'));
      const labels = rows.map((row) => row.textContent?.trim() ?? "").filter(Boolean);

      results.push({
        ratio,
        scrollTop: Math.round(grid.scrollTop),
        rowCount: rows.length,
        nonBlankCount: labels.length,
        first: labels[0] ?? "",
        last: labels[labels.length - 1] ?? "",
      });
    }

    return {
      scrollHeight: grid.scrollHeight,
      clientHeight: grid.clientHeight,
      results,
    };
  });
}

async function runManualBrowserJourney(page: Page, testInfo: TestInfo) {
  const consoleProblems: string[] = [];
  const treeResponses: number[] = [];
  const failedRequests: string[] = [];

  page.on("console", (message) => {
    if (message.type() === "error" || message.type() === "warning") {
      consoleProblems.push(`${message.type()}: ${message.text()}`);
    }
  });
  page.on("response", (response) => {
    if (TREE_API_PATTERN.test(response.url())) {
      treeResponses.push(response.status());
    }
  });
  page.on("requestfailed", (request) => {
    const failureText = request.failure()?.errorText ?? "";
    if (TREE_API_PATTERN.test(request.url()) && failureText !== "net::ERR_ABORTED") {
      failedRequests.push(`${request.url()} ${failureText}`);
    }
  });

  const metrics = await waitForTreeTable(page);
  await expect(page.locator('[data-testid="total-count"]')).toHaveText("1,000,000");
  expect(metrics.apiMs).toBeLessThan(MAX_INITIAL_TREE_API_MS);
  expect(metrics.renderMs).toBeLessThan(MAX_INITIAL_RENDER_MS);
  await expectVisibleRowsAreNotBlank(page);

  await page.getByRole("button", { name: "すべて展開" }).click();
  await expect(page.getByRole("button", { name: /を折りたたむ/ }).first()).toBeVisible();
  await expect(page.getByText("表示行")).toBeVisible();

  const scrollCheck = await scrollGridAndCollectRows(page);
  expect(scrollCheck.scrollHeight).toBeGreaterThan(scrollCheck.clientHeight);
  for (const result of scrollCheck.results) {
    expect(result.rowCount).toBeGreaterThan(0);
    expect(result.nonBlankCount).toBe(result.rowCount);
    expect(result.first.length).toBeGreaterThan(0);
    expect(result.last.length).toBeGreaterThan(0);
  }

  const amountHeader = page.getByRole("columnheader", { name: /金額/ });
  const sortResponsePromise = page.waitForResponse((response) =>
    TREE_API_PATTERN.test(response.url()),
  );
  await amountHeader.click();
  const sortResponse = await sortResponsePromise;
  expect(sortResponse.ok()).toBeTruthy();
  await expect(page.getByRole("columnheader", { name: /金額 ▲/ })).toBeVisible();
  await expectVisibleRowsAreNotBlank(page);

  const pageResponsePromise = page.waitForResponse((response) =>
    TREE_API_PATTERN.test(response.url()),
  );
  await page.getByLabel("次のページ").click();
  const pageResponse = await pageResponsePromise;
  expect(pageResponse.ok()).toBeTruthy();
  await expect(page.getByRole("button", { name: "2" })).toHaveAttribute("aria-current", "page");
  await expectVisibleRowsAreNotBlank(page);

  await page.screenshot({
    path: testInfo.outputPath("tree-table-browser-journey.png"),
    fullPage: false,
  });

  expect(treeResponses.some((status) => status === 200)).toBeTruthy();
  expect(failedRequests).toEqual([]);
  expect(consoleProblems).toEqual([]);

  return { metrics, scrollCheck, treeResponses };
}

test.describe("BigTreeTable E2E", () => {
  test("100万件データをTreeTableとして初期表示でき、初期描画が遅すぎない", async ({ page }) => {
    const metrics = await waitForTreeTable(page);

    await expect(page.locator('[data-testid="total-count"]')).toHaveText("1,000,000");
    expect(metrics.apiMs).toBeLessThan(MAX_INITIAL_TREE_API_MS);
    expect(metrics.renderMs).toBeLessThan(MAX_INITIAL_RENDER_MS);
    await expectVisibleRowsAreNotBlank(page);
  });

  test("展開後に仮想スクロールしてもブランク表示にならない", async ({ page }) => {
    await waitForTreeTable(page);

    await page.getByRole("button", { name: "すべて展開" }).click();
    await expect(page.locator('[role="row"][aria-rowindex]').first()).toBeVisible();

    const grid = page.locator('[role="grid"]');
    await expect(grid).toBeVisible();

    for (const ratio of [0, 0.25, 0.5, 0.75, 1]) {
      await grid.evaluate((element, scrollRatio) => {
        element.scrollTop = (element.scrollHeight - element.clientHeight) * scrollRatio;
      }, ratio);
      await page.waitForFunction(() => {
        const rows = document.querySelectorAll('[role="row"][aria-rowindex]');
        return rows.length > 0 && Array.from(rows).some((row) => row.textContent?.trim());
      });
      await expectVisibleRowsAreNotBlank(page);
    }
  });

  test("ソートとページング後もデータ遅延なく行が戻る", async ({ page }) => {
    await waitForTreeTable(page);

    const amountHeader = page.getByRole("columnheader", { name: /金額/ });
    const responsePromise = page.waitForResponse((response) =>
      TREE_API_PATTERN.test(response.url()),
    );

    await amountHeader.click();
    const response = await responsePromise;
    expect(response.ok()).toBeTruthy();
    await expect(amountHeader).toHaveAttribute("aria-sort", "ascending");
    await expect(page.locator('[role="row"][aria-rowindex]').first()).toBeVisible();
    await expectVisibleRowsAreNotBlank(page);

    const nextResponsePromise = page.waitForResponse((nextResponse) =>
      TREE_API_PATTERN.test(nextResponse.url()),
    );
    await page.getByLabel("次のページ").click();
    const nextResponse = await nextResponsePromise;
    expect(nextResponse.ok()).toBeTruthy();
    await expect(page.getByRole("button", { name: "2" })).toHaveAttribute("aria-current", "page");
    await expectVisibleRowsAreNotBlank(page);
  });

  test("ブラウザ操作で初期表示、展開、スクロール、ソート、ページングを確認する", async ({
    page,
  }, testInfo) => {
    const result = await runManualBrowserJourney(page, testInfo);

    await testInfo.attach("manual-browser-journey-metrics", {
      body: JSON.stringify(result, null, 2),
      contentType: "application/json",
    });
  });
});
