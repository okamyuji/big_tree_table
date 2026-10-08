// @vitest-environment node
import { execFileSync } from "node:child_process";
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import {
  findPolicyProblems,
  findToolchainProblems,
} from "../../scripts/check-dependency-policy.mjs";

const CORE = "@voidzero-dev/vite-plus-core";
const vitePlus = { version: "1.0.0", dependencies: { vitest: "5.0.1" } };
const strictPolicy = {
  minimumReleaseAge: 10080,
  minimumReleaseAgeExclude: [],
  strictPeerDependencies: true,
  peerDependencyRules: { allowedVersions: { vite: "1" } },
};

describe("findPolicyProblems", () => {
  it("accepts the strict policy", () => {
    expect(findPolicyProblems(strictPolicy)).toEqual([]);
  });

  it("accepts a release age exactly at the 7-day minimum and above it", () => {
    expect(findPolicyProblems({ ...strictPolicy, minimumReleaseAge: 10080 })).toEqual([]);
    expect(findPolicyProblems({ ...strictPolicy, minimumReleaseAge: 20160 })).toEqual([]);
  });

  it("rejects a release age below 7 days", () => {
    expect(findPolicyProblems({ ...strictPolicy, minimumReleaseAge: 10079 })).toEqual([
      "minimumReleaseAge is 10079, expected >= 10080",
    ]);
  });

  it("rejects a missing release age", () => {
    expect(findPolicyProblems({ ...strictPolicy, minimumReleaseAge: undefined })).toEqual([
      "minimumReleaseAge is undefined, expected >= 10080",
    ]);
  });

  it("accepts version-pinned excludes, including scoped packages", () => {
    const excludes = ["source-map-js@1.2.2", "@voidzero-dev/vite-plus-core@1.1.0"];

    expect(findPolicyProblems({ ...strictPolicy, minimumReleaseAgeExclude: excludes })).toEqual([]);
  });

  it("rejects package-wide excludes such as the ones vp migrate adds", () => {
    const excludes = ["vite-plus", "@voidzero-dev/*", "vitest@*"];

    expect(findPolicyProblems({ ...strictPolicy, minimumReleaseAgeExclude: excludes })).toEqual([
      'minimumReleaseAgeExclude "vite-plus" must pin a version (name@x.y.z)',
      'minimumReleaseAgeExclude "@voidzero-dev/*" must pin a version (name@x.y.z)',
      'minimumReleaseAgeExclude "vitest@*" must pin a version (name@x.y.z)',
    ]);
  });

  it("rejects non-strict peer dependencies", () => {
    expect(findPolicyProblems({ ...strictPolicy, strictPeerDependencies: false })).toEqual([
      "strictPeerDependencies must be true",
    ]);
  });

  it("rejects allowAny and wildcard allowed versions", () => {
    const peerDependencyRules = { allowedVersions: { vite: "*" }, allowAny: ["vite"] };

    expect(findPolicyProblems({ ...strictPolicy, peerDependencyRules })).toEqual([
      'peerDependencyRules.allowAny "vite" must be removed',
      'peerDependencyRules.allowedVersions.vite must not be "*"',
    ]);
  });

  it("treats absent exclude and peer settings as empty", () => {
    expect(
      findPolicyProblems({ minimumReleaseAge: 10080, strictPeerDependencies: true }),
    ).toEqual([]);
  });
});

describe("findToolchainProblems", () => {
  const aligned = {
    vitePlus,
    vite: { name: CORE, version: "1.0.0" },
    usedVitest: { version: "5.0.1" },
    rootVitest: { version: "5.0.1" },
  };

  it("accepts vite and vitest that match vite-plus", () => {
    expect(findToolchainProblems(aligned)).toEqual([]);
  });

  it("accepts a project without a root vitest", () => {
    expect(findToolchainProblems({ ...aligned, rootVitest: undefined })).toEqual([]);
  });

  it("reports vite-plus-core lagging behind vite-plus", () => {
    const problems = findToolchainProblems({
      ...aligned,
      vitePlus: { version: "1.1.0", dependencies: { vitest: "5.0.1" } },
    });

    expect(problems).toEqual([`vite is ${CORE}@1.0.0, expected ${CORE}@1.1.0`]);
  });

  it("reports upstream vite installed instead of the vite-plus-core alias", () => {
    const problems = findToolchainProblems({ ...aligned, vite: { name: "vite", version: "1.0.0" } });

    expect(problems).toEqual([`vite is vite@1.0.0, expected ${CORE}@1.0.0`]);
  });

  it("reports a stale override that makes vite-plus load an old vitest", () => {
    const problems = findToolchainProblems({
      ...aligned,
      usedVitest: { version: "5.0.0" },
      rootVitest: undefined,
    });

    expect(problems).toEqual([
      "vitest used by vite-plus is 5.0.0, expected 5.0.1 (bundled with vite-plus 1.0.0)",
    ]);
  });

  it("reports a root vitest that splits from the bundled one", () => {
    const problems = findToolchainProblems({ ...aligned, rootVitest: { version: "5.0.3" } });

    expect(problems).toEqual([
      "root vitest is 5.0.3, expected 5.0.1 (bundled with vite-plus 1.0.0)",
    ]);
  });
});

describe("check-dependency-policy CLI", () => {
  const script = fileURLToPath(new URL("../../scripts/check-dependency-policy.mjs", import.meta.url));
  const { packageManager } = JSON.parse(
    readFileSync(fileURLToPath(new URL("../../package.json", import.meta.url)), "utf8"),
  ) as { packageManager: string };

  const workspaceYaml = (extra = "") =>
    `minimumReleaseAge: 10080\nstrictPeerDependencies: true\n${extra}`;

  // 偽の node_modules と pnpm-workspace.yaml を置いた一時ディレクトリでスクリプトを動かす。
  const runInFakeProject = (
    workspace: string,
    packages: Record<string, object>,
    run: (cli: string) => void,
  ) => {
    const root = mkdtempSync(join(tmpdir(), "dependency-policy-"));
    try {
      mkdirSync(join(root, "scripts"));
      const cli = join(root, "scripts", "check-dependency-policy.mjs");
      copyFileSync(script, cli);
      writeFileSync(join(root, "package.json"), JSON.stringify({ name: "fake", private: true, packageManager }));
      writeFileSync(join(root, "pnpm-workspace.yaml"), workspace);
      for (const [name, pkg] of Object.entries(packages)) {
        mkdirSync(join(root, "node_modules", name), { recursive: true });
        writeFileSync(join(root, "node_modules", name, "package.json"), JSON.stringify(pkg));
      }
      run(cli);
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  };

  const runCli = (cli: string) => {
    try {
      execFileSync(process.execPath, [cli], { stdio: "pipe" });
      return { status: 0, stderr: "" };
    } catch (error) {
      const { status, stderr } = error as { status: number; stderr: Buffer };
      return { status, stderr: stderr.toString() };
    }
  };

  const alignedPackages = {
    "vite-plus": vitePlus,
    vite: { name: CORE, version: "1.0.0" },
    vitest: { version: "5.0.1" },
  };

  it("exits 0 for the installed toolchain and workspace", () => {
    expect(runCli(script)).toEqual({ status: 0, stderr: "" });
  });

  it("exits 0 for an aligned fake project", () => {
    runInFakeProject(workspaceYaml(), alignedPackages, (cli) => {
      expect(runCli(cli)).toEqual({ status: 0, stderr: "" });
    });
  });

  it("exits 1 on a single problem", () => {
    runInFakeProject(
      workspaceYaml("peerDependencyRules:\n  allowAny:\n    - vite\n"),
      alignedPackages,
      (cli) => {
        const { status, stderr } = runCli(cli);

        expect(status).toBe(1);
        expect(stderr).toContain('peerDependencyRules.allowAny "vite" must be removed');
      },
    );
  });

  it("exits 1 and lists the problems after an unreviewed vp migrate", () => {
    runInFakeProject(
      workspaceYaml("minimumReleaseAgeExclude:\n  - vite-plus\npeerDependencyRules:\n  allowAny:\n    - vite\n"),
      { ...alignedPackages, vitest: { version: "5.0.3" } },
      (cli) => {
        const { status, stderr } = runCli(cli);

        expect(status).toBe(1);
        expect(stderr).toContain("pnpm exec vp migrate");
        expect(stderr).toContain('minimumReleaseAgeExclude "vite-plus" must pin a version');
        expect(stderr).toContain('peerDependencyRules.allowAny "vite" must be removed');
        expect(stderr).toContain("vitest used by vite-plus is 5.0.3, expected 5.0.1");
      },
    );
  });
});
