import { Container, dag } from './../../../../../api/client.gen.js';
import { func, object } from "../../../../decorators/index.js"

/**
 * Example of iface
 */
export interface IFace {
  /**
   * Foo fct
   */
  foo: () => string
  count: (n: number) => number[]
  withClass: (c: Container) => Promise<string>
  outClass: () => Container
  self(): IFace
  withSelf: (i: IFace) => void
}

@object()
export class Interface {
  @func()
  hello(): string {
    return "hello"
  }

  @func()
  iface(): IFace {
    const iface = {
      foo: () => "",
      count: (n: number) => [4],
      withClass: async (c: Container) => await c.id(),
      outClass: () => dag.container(),
      self: () => iface,
      withSelf: (i: IFace) => {},
    }

    return iface
  }
}
