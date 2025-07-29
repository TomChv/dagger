// THIS FILE IS AUTO GENERATED. PLEASE DO NOT EDIT.
import { typeDefEntrypoint } from "@dagger.io/dagger"
import * as fs from "fs"
import * as path from "path"

const allowedExtensions = [".ts", ".mts"]

function listTsFilesInModule(dir = import.meta.dirname): string[] {
  let bundle = true

  // For background compatibility, if there's a package.json in the sdk directory
  // We should set the right path to the client.
  if (fs.existsSync(`${import.meta.dirname}/../sdk/package.json`)) {
    bundle = false
  }

  const res = fs.readdirSync(dir).map((file) => {
    const filepath = path.join(dir, file)

    const stat = fs.statSync(filepath)

    if (stat.isDirectory()) {
      return listTsFilesInModule(filepath)
    }

    const ext = path.extname(filepath)
    if (allowedExtensions.find((allowedExt) => allowedExt === ext)) {
      return [path.join(dir, file)]
    }

    return []
  })

  return res.reduce(
    (p, c) => [...c, ...p],
    [`${import.meta.dirname}/../sdk/${bundle ? "" : "src/api/"}client.gen.ts`],
  )
}

const files = listTsFilesInModule()

const help = `Usage: __typedef.entrypoint.ts --output-file=string> --module-name=string`
const args = process.argv.slice(2)

class Arg<T> {
  constructor(
    public name: string,
    public value: T | null,
  ) {}
}

const outputFile = new Arg<string>("output-file", null)
const moduleName = new Arg<string>("module-name", null)

console.log("parsing args", args)

// Parse arguments from the CLI
for (const arg of args) {
  const [name, value] = arg.slice("--".length).split("=")
  switch (name) {
    case "output-file":
      if (value === undefined) {
        console.error(`Missing value for ${name}\n ${help}`)
        process.exit(1)
      }

      outputFile.value = value

      break
    case "module-name":
      if (value === undefined) {
        console.error(`Missing value for ${name}\n ${help}`)
        process.exit(1)
      }

      moduleName.value = value

      break
  }
}

console.log("calling entrypoints", files, outputFile.value, moduleName.value)

typeDefEntrypoint(files, outputFile.value, moduleName.value)
