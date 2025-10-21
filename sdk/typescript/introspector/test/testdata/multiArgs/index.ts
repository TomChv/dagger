import { func, object } from "@dagger.io/dagger"
import type { float } from "@dagger.io/dagger"

@object()
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export class MultiArgs {
  @func()
  compute(a: number, b: number, c: float): float {
    return a * b + c
  }
}
