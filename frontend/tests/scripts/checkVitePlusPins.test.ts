// @vitest-environment node
import { execFileSync } from "node:child_process";
import { copyFileSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { findMisalignedPins } from "../../scripts/check-vite-plus-pins.mjs";

const CORE = "@voidzero-dev/vite-plus-core";
const vitePlus = { version: "1.0.0", dependencies: { vitest: "5.0.1" } };

describe("findMisalignedPins", () => {
  it("returns no problems when vite and vitest match vite-plus", () => {
    const problems = findMisalignedPins({
      vitePlus,
      vite: { name: CORE, version: "1.0.0" },
      vitest: { version: "5.0.1" },
    });

    expect(problems).toEqual([]);
  });

  it("reports vite-plus-core lagging behind vite-plus", () => {
    const problems = findMisalignedPins({
      vitePlus: { version: "1.1.0", dependencies: { vitest: "5.0.1" } },
      vite: { name: CORE, version: "1.0.0" },
      vitest: { version: "5.0.1" },
    });

    expect(problems).toEqual([`vite is ${CORE}@1.0.0, expected ${CORE}@1.1.0`]);
  });

  it("reports upstream vite installed instead of the vite-plus-core alias", () => {
    const problems = findMisalignedPins({
      vitePlus,
      vite: { name: "vite", version: "1.0.0" },
      vitest: { version: "5.0.1" },
    });

    expect(problems).toEqual([`vite is vite@1.0.0, expected ${CORE}@1.0.0`]);
  });

  it("reports a vitest pin that differs from the bundled version", () => {
    const problems = findMisalignedPins({
      vitePlus,
      vite: { name: CORE, version: "1.0.0" },
      vitest: { version: "5.0.3" },
    });

    expect(problems).toEqual([
      "vitest is 5.0.3, expected 5.0.1 (bundled with vite-plus 1.0.0)",
    ]);
  });

  it("reports vitest when vite-plus declares no bundled vitest", () => {
    const problems = findMisalignedPins({
      vitePlus: { version: "1.0.0" },
      vite: { name: CORE, version: "1.0.0" },
      vitest: { version: "5.0.1" },
    });

    expect(problems).toEqual([
      "vitest is 5.0.1, expected undefined (bundled with vite-plus 1.0.0)",
    ]);
  });

  it("skips vitest when it is not installed at the project root", () => {
    const problems = findMisalignedPins({
      vitePlus,
      vite: { name: CORE, version: "1.0.0" },
      vitest: undefined,
    });

    expect(problems).toEqual([]);
  });

  it("reports both pins when both drift", () => {
    const problems = findMisalignedPins({
      vitePlus: { version: "1.1.0", dependencies: { vitest: "5.0.3" } },
      vite: { name: CORE, version: "1.0.0" },
      vitest: { version: "5.0.1" },
    });

    expect(problems).toHaveLength(2);
  });
});

describe("check-vite-plus-pins CLI", () => {
  const script = fileURLToPath(new URL("../../scripts/check-vite-plus-pins.mjs", import.meta.url));

  it("exits 0 for the installed toolchain", () => {
    expect(() => execFileSync(process.execPath, [script], { stdio: "pipe" })).not.toThrow();
  });

  // 偽の node_modules を置いた一時ディレクトリでスクリプトを動かす。
  const runInFakeProject = (packages: Record<string, object>, run: (cli: string) => void) => {
    const root = mkdtempSync(join(tmpdir(), "vite-plus-pins-"));
    try {
      mkdirSync(join(root, "scripts"));
      const cli = join(root, "scripts", "check-vite-plus-pins.mjs");
      copyFileSync(script, cli);
      for (const [name, pkg] of Object.entries(packages)) {
        mkdirSync(join(root, "node_modules", name), { recursive: true });
        writeFileSync(join(root, "node_modules", name, "package.json"), JSON.stringify(pkg));
      }
      run(cli);
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  };

  it("exits 0 when vitest is not installed at the project root", () => {
    runInFakeProject(
      { "vite-plus": vitePlus, vite: { name: CORE, version: "1.0.0" } },
      (cli) => {
        expect(() => execFileSync(process.execPath, [cli], { stdio: "pipe" })).not.toThrow();
      },
    );
  });

  it("exits 1 and names vp migrate when an installed pin drifts", () => {
    runInFakeProject(
      {
        "vite-plus": vitePlus,
        vite: { name: CORE, version: "1.0.0" },
        vitest: { version: "5.0.3" },
      },
      (cli) => {
        let failure: { status: number; stderr: Buffer } | undefined;
        try {
          execFileSync(process.execPath, [cli], { stdio: "pipe" });
        } catch (error) {
          failure = error as { status: number; stderr: Buffer };
        }

        expect(failure?.status).toBe(1);
        expect(failure?.stderr.toString()).toContain("vitest is 5.0.3, expected 5.0.1");
        expect(failure?.stderr.toString()).toContain("pnpm exec vp migrate");
      },
    );
  });
});
