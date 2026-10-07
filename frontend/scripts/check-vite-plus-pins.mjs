// Vite+ は vite-plus、vite の別名（vite-plus-core）、vitest の版をそろえて上げる必要がある。
// Dependabot は1つずつ上げるので、ずれたまま CI を通さないよう、入った版を突き合わせる。
// https://viteplus.dev/guide/upgrade-project
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const CORE = "@voidzero-dev/vite-plus-core";

export function findMisalignedPins({ vitePlus, vite, vitest }) {
  const problems = [];
  if (vite.name !== CORE || vite.version !== vitePlus.version) {
    problems.push(
      `vite is ${vite.name}@${vite.version}, expected ${CORE}@${vitePlus.version}`,
    );
  }
  const bundled = vitePlus.dependencies?.vitest;
  if (vitest.version !== bundled) {
    problems.push(
      `vitest is ${vitest.version}, expected ${bundled} (bundled with vite-plus ${vitePlus.version})`,
    );
  }
  return problems;
}

function readInstalled(name) {
  const url = new URL(`../node_modules/${name}/package.json`, import.meta.url);
  return JSON.parse(readFileSync(url, "utf8"));
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const problems = findMisalignedPins({
    vitePlus: readInstalled("vite-plus"),
    vite: readInstalled("vite"),
    vitest: readInstalled("vitest"),
  });
  if (problems.length > 0) {
    console.error([...problems, "Run `pnpm exec vp migrate` to realign them."].join("\n"));
    process.exit(1);
  }
}
