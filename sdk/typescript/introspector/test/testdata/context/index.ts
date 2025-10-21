import { Directory, func, object, argument } from "@dagger.io/dagger"

@object()
export class Context {
  @func()
  helloWorld(
    @argument({ defaultPath: "." })
    dir: Directory,
  ): string {
    return `hello ${name}`
  }

  @func()
  helloWorldIgnored(
    @argument({ defaultPath: ".", ignore: ["dir"] })
    dir: Directory,
  ): string {
    return `hello ${name}`
  }
}
