import { TypeDefKind } from "@dagger.io/dagger"

import type { TypeDef } from "../typedef/typedef"
import { DaggerConstructor } from "./constructor.js"
import type { DaggerFunctions } from "./function.js"
import { Locatable } from "./locatable.js"
import type { References } from "./reference.js"

export interface DaggerObjectPropertyBase extends Locatable {
  name: string
  description: string
  alias?: string
  isExposed: boolean
  type?: TypeDef<TypeDefKind>

  propagateReferences(references: References): void
}

export type DaggerObjectPropertiesBase = {
  [name: string]: DaggerObjectPropertyBase
}

export interface DaggerObjectBase extends Locatable {
  name: string
  description: string
  _constructor: DaggerConstructor | undefined
  methods: DaggerFunctions
  properties: DaggerObjectPropertiesBase

  kind(): "class" | "object"

  propagateReferences(references: References): void
}

export type DaggerObjectsBase = { [name: string]: DaggerObjectBase }
