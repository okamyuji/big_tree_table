// Vite+ は vite-plus、vite の別名（vite-plus-core）、vitest の版をそろえて上げる必要がある。
// Dependabot は1つずつ上げるので、ずれたまま CI を通さないよう、入った版を突き合わせる。
// https://viteplus.dev/guide/upgrade-project
import { existsSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const CORE = "@voidzero-dev/vite-plus-core";

export function findMisalignedPins({ vitePlus, vite, vitest }) {
  const problems = [];
  if (vite.name !== CORE || vite.version !== vitePlus.version) {
    problems.push(
      `vite is ${vite.name}@${vite.version}, expected ${CORE}@${vitePlus.version}`,
    );
  }
  // ルートに vitest が無ければ、Vitest は vite-plus が持つ1組だけで、分かれようがない。
  const bundled = vitePlus.dependencies?.vitest;
  if (vitest && vitest.version !== bundled) {
    problems.push(
      `vitest is ${vitest.version}, expected ${bundled} (bundled with vite-plus ${vitePlus.version})`,
    );
  }
  return problems;
}

function readInstalled(name, { optional = false } = {}) {
  const url = new URL(`../node_modules/${name}/package.json`, import.meta.url);
  if (optional && !existsSync(url)) return undefined;
  return JSON.parse(readFileSync(url, "utf8"));
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const problems = findMisalignedPins({
    vitePlus: readInstalled("vite-plus"),
    vite: readInstalled("vite"),
    vitest: readInstalled("vitest", { optional: true }),
  });
  if (problems.length > 0) {
    console.error(
      [
        ...problems,
        "Run `pnpm exec vp migrate` to realign them, then revert the minimumReleaseAgeExclude",
        "and peerDependencyRules.allowAny entries it adds to pnpm-workspace.yaml.",
      ].join("\n"),
    );
    process.exit(1);
  }
}
