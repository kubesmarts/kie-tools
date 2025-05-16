#!/usr/bin/env node
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const dirsToScan = ["packages", "examples"];

const readJson = (p) => JSON.parse(fs.readFileSync(p, "utf8"));

// Map: name -> package path (e.g., packages/foo or examples/bar)
const packageMap = {};

for (const base of dirsToScan) {
  const fullBasePath = path.join(repoRoot, base);
  if (!fs.existsSync(fullBasePath)) continue;
  fs.readdirSync(fullBasePath).forEach((pkg) => {
    const pkgPath = path.join(fullBasePath, pkg, "package.json");
    if (fs.existsSync(pkgPath)) {
      const json = readJson(pkgPath);
      packageMap[json.name] = path.join(base, pkg);
    }
  });
}

// Recursively walk dependencies
const visited = new Set();
function walkDeps(pkgName) {
  if (visited.has(pkgName)) return;
  visited.add(pkgName);
  const pkgJsonPath = path.join(repoRoot, packageMap[pkgName], "package.json");
  const json = readJson(pkgJsonPath);
  const deps = {
    ...json.dependencies,
    ...json.devDependencies,
    ...json.optionalDependencies,
  };
  Object.keys(deps || {}).forEach((dep) => {
    if (packageMap[dep]) walkDeps(dep); // only follow local packages
  });
}

// 🌱 Start from your root entry points
const entryPoints = [
  "@kie-tools/kn-plugin-workflow",
  "@osl/osl-operator-bundle-image",
  "@kie-tools/sonataflow-management-console-image",
  "@osl/osl-data-index-ephemeral-image",
  "@osl/osl-data-index-postgresql-image",
  "@osl/osl-db-migrator-tool-image",
  "@osl/osl-jobs-service-ephemeral-image",
  "@osl/osl-jobs-service-postgresql-image",
  "@osl/osl-management-console-image",
  "@osl/osl-swf-builder-image",
  "@osl/osl-swf-devmode-image",
];

entryPoints.forEach((pkg) => {
  if (!packageMap[pkg]) {
    console.warn(`⚠️ Warning: Entry point ${pkg} not found in workspace`);
  } else {
    walkDeps(pkg);
  }
});

// ✅ Result
const used = [...visited];
const all = Object.keys(packageMap);
const unused = all.filter((pkg) => !visited.has(pkg));

console.log("✅ Used workspace packages:");
used.forEach((pkg) => console.log("  -", pkg));

console.log("\n🗑️ Unused workspace packages:");
unused.forEach((pkg) => console.log("  -", pkg));

console.log("\n🧹 Copy and run to delete:");
const deleteCmd = "rm -rf " + unused.map((pkg) => packageMap[pkg]).join(" ");
console.log(deleteCmd);
