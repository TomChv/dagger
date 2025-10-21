import { AST } from "../ast"
import { DaggerModule } from "../module"
import * as strcase from "../utils/strcase"

export async function scan(files: string[], moduleName = "") {
  if (files.length === 0) {
    throw new Error("no files to introspect found")
  }

  moduleName = strcase.PascalCase(moduleName)
  const ast = new AST(files, [])

  return new DaggerModule(moduleName, [], ast)
}
