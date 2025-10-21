import assert from "assert"
import * as path from "path"
import { fileURLToPath } from "url"

import * as fsutil from "../src/utils/fsutil"

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const rootDirectory = `${__dirname}/testdata`

describe("ListFiles", function () {
  it("Should correctly list files from hello world example and ignore unwanted files", async function () {
    const files = await fsutil.listTsFilesInModule(
      `${rootDirectory}/helloWorld`,
    )

    assert.deepEqual(
      files.map((f) => path.basename(f)),
      ["helloWorld.ts"],
    )
  })
})
