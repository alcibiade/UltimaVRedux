import { spawnSync } from "node:child_process"
import { fileURLToPath } from "node:url"
import { resolve } from "node:path"
import ts from "typescript"

const directory = fileURLToPath(new URL(".", import.meta.url))
const projects = ["app", "root-tests", "component-tests", "browser"]

function configuration(name) {
  const path = resolve(directory, name)
  const config = ts.readConfigFile(path, ts.sys.readFile)
  if (config.error) throw new Error(ts.flattenDiagnosticMessageText(config.error.messageText, "\n"))
  // TypeScript's normal inventory omits Vue roots; vue-tsc checks those too.
  const host = { ...ts.sys, readDirectory: (root, extensions, ...rest) => ts.sys.readDirectory(root, [...extensions, ".vue"], ...rest) }
  const parsed = ts.parseJsonConfigFileContent(config.config, host, directory)
  if (parsed.errors.length) throw new Error(parsed.errors.map((error) => ts.flattenDiagnosticMessageText(error.messageText, "\n")).join("\n"))
  return parsed
}

export function verifyPartition(expected, partitions) {
  const remaining = new Set(expected)
  for (const [name, files] of partitions) {
    if (!files.length) throw new Error(`Empty typecheck partition: ${name}`)
    for (const file of files) {
      if (!remaining.delete(file)) throw new Error(`Duplicate or unexpected typecheck root in ${name}: ${file}`)
    }
  }
  if (remaining.size) throw new Error(`Missing typecheck roots: ${[...remaining].join(", ")}`)
}

function main() {
  const canonical = configuration("tsconfig.json")
  const partitions = projects.map((name) => [name, configuration(`tsconfig.${name}.json`)])
  verifyPartition(canonical.fileNames, partitions.map(([name, config]) => [name, config.fileNames]))
  for (const [name, config] of partitions) {
    if (JSON.stringify(config.options) !== JSON.stringify(canonical.options)) {
      throw new Error(`Typecheck options differ from tsconfig.json: ${name}`)
    }
  }
  console.log(`Typecheck inventory: ${canonical.fileNames.length} roots, each assigned once; imported dependencies remain checked.`)
  for (const [name, config] of partitions) {
    console.log(`Typecheck ${name}: ${config.fileNames.length} roots`)
    // Bound the working heap, not the OS footprint. Each checker releases its
    // complete type graph before the next starts; external RSS supervision is
    // still required on memory-constrained hosts.
    const result = spawnSync(process.execPath, [
      "--max-old-space-size=512", "--max-semi-space-size=8",
      "node_modules/vue-tsc/bin/vue-tsc.js", "--noEmit", "-p", `tsconfig.${name}.json`,
    ], { cwd: directory, stdio: "inherit" })
    if (result.error) throw result.error
    if (result.status !== 0) throw new Error(`Typecheck ${name} failed (${result.signal ?? result.status})`)
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main()
