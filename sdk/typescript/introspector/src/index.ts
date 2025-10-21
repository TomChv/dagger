import { listTsFilesInModule } from "./utils/fsutil"

async function main() {
  const dirSourcePath = process.env.MODULE_SOURCE_PATH ?? "."
  const files = listTsFilesInModule(dirSourcePath)

  
}

main()
