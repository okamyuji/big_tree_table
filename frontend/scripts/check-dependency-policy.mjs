// 依存の守り（7日の待ち期間、strict peer）と Vite+ の版のそろいを lint で確かめる。
// vp migrate などの道具はこれらを緩めることがあり、人の目では戻し忘れるため、機械で止める。
// https://viteplus.dev/guide/upgrade-project
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, realpathSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const CORE = "@voidzero-dev/vite-plus-core";
const MIN_RELEASE_AGE_MINUTES = 10080;
// 例外は版を指定したもの（name@1.2.3 など）だけを許す。名前だけの例外は全版の待ち期間を外す。
const VERSIONED_EXCLUDE = /^(@[^/@]+\/)?[^/@]+@[^*]+$/;

export function findPolicyProblems({
  minimumReleaseAge,
  minimumReleaseAgeExclude = [],
  strictPeerDependencies,
  peerDependencyRules = {},
}) {
  const problems = [];
  if (!(Number(minimumReleaseAge) >= MIN_RELEASE_AGE_MINUTES)) {
    problems.push(`minimumReleaseAge is ${minimumReleaseAge}, expected >= ${MIN_RELEASE_AGE_MINUTES}`);
  }
  for (const entry of minimumReleaseAgeExclude) {
    if (!VERSIONED_EXCLUDE.test(entry)) {
      problems.push(`minimumReleaseAgeExclude "${entry}" must pin a version (name@x.y.z)`);
    }
  }
  if (strictPeerDependencies !== true) {
    problems.push("strictPeerDependencies must be true");
  }
  for (const name of peerDependencyRules.allowAny ?? []) {
    problems.push(`peerDependencyRules.allowAny "${name}" must be removed`);
  }
  for (const [name, range] of Object.entries(peerDependencyRules.allowedVersions ?? {})) {
    if (range === "*") {
      problems.push(`peerDependencyRules.allowedVersions.${name} must not be "*"`);
    }
  }
  return problems;
}

// usedVitest は vite-plus が実際に読み込む vitest、rootVitest はプロジェクト直下の vitest。
export function findToolchainProblems({ vitePlus, vite, usedVitest, rootVitest }) {
  const problems = [];
  if (vite.name !== CORE || vite.version !== vitePlus.version) {
    problems.push(`vite is ${vite.name}@${vite.version}, expected ${CORE}@${vitePlus.version}`);
  }
  const bundled = vitePlus.dependencies?.vitest;
  for (const [label, vitest] of [["vitest used by vite-plus", usedVitest], ["root vitest", rootVitest]]) {
    if (vitest && vitest.version !== bundled) {
      problems.push(`${label} is ${vitest.version}, expected ${bundled} (bundled with vite-plus ${vitePlus.version})`);
    }
  }
  return problems;
}

function readJson(path) {
  return existsSync(path) ? JSON.parse(readFileSync(path, "utf8")) : undefined;
}

function pnpmConfig(key, cwd) {
  const out = execFileSync("pnpm", ["config", "get", key, "--json"], { cwd, encoding: "utf8" }).trim();
  return out === "" || out === "undefined" ? undefined : JSON.parse(out);
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const root = join(dirname(fileURLToPath(import.meta.url)), "..");
  const modules = join(root, "node_modules");
  const vitePlusDir = realpathSync(join(modules, "vite-plus"));
  const problems = [
    ...findPolicyProblems({
      minimumReleaseAge: pnpmConfig("minimumReleaseAge", root),
      minimumReleaseAgeExclude: pnpmConfig("minimumReleaseAgeExclude", root),
      strictPeerDependencies: pnpmConfig("strictPeerDependencies", root),
      peerDependencyRules: pnpmConfig("peerDependencyRules", root),
    }),
    ...findToolchainProblems({
      vitePlus: readJson(join(vitePlusDir, "package.json")),
      vite: readJson(join(modules, "vite", "package.json")),
      usedVitest: readJson(join(vitePlusDir, "..", "vitest", "package.json")),
      rootVitest: readJson(join(modules, "vitest", "package.json")),
    }),
  ];
  if (problems.length > 0) {
    console.error(
      [
        "Dependency policy check failed. Realign Vite+ with `pnpm exec vp migrate`,",
        "then restore the policy entries in pnpm-workspace.yaml:",
        ...problems,
      ].join("\n"),
    );
    process.exit(1);
  }
}
