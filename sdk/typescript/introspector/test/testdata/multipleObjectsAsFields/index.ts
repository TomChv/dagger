import { object, func } from "@dagger.io/dagger"

import { Lint } from "./lint.js"
import { Test } from "./test.js"

@object()
export class MultipleObjectsAsFields {
  @func()
  test: Test = new Test()

  @func()
  lint: Lint = new Lint()

  constructor() {}
}
