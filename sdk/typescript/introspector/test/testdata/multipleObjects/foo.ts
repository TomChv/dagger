/**
 * Foo object module
 *
 * Compose of bar but its file description should be ignore.
 */
import { func, object } from "@dagger.io/dagger"

import { Bar } from "./bar.js"

/**
 * Foo class
 */
@object()
export class MultipleObjects {
  /**
   * Return Bar object
   */
  @func()
  bar(): Bar {
    return new Bar()
  }
}
