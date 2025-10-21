import { func, object } from "@dagger.io/dagger"

@object()
export class Test {
  @func()
  echo(): string {
    return "world"
  }
}
