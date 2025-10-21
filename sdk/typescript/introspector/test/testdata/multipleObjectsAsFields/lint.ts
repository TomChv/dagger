import { func, object } from "@dagger.io/dagger"

@object()
export class Lint {
  @func()
  echo(): string {
    return "world"
  }
}
