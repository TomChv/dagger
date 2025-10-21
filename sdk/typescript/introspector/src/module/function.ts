import { TypeDefKind } from "@dagger.io/dagger"
import ts from "typescript"

import { AST, isTypeDefResolved, resolveTypeDef } from "../ast"
import type { TypeDef } from "../typedef/typedef"
import { FUNCTION_DECORATOR } from "../utils/decorator"
import { DaggerArgument } from "./argument.js"
import type { DaggerArguments } from "./argument.js"
import { Locatable } from "./locatable.js"
import type { References } from "./reference"

export type DaggerFunctions = { [name: string]: DaggerFunction }

export class DaggerFunction extends Locatable {
  public name: string
  public description: string
  private _returnTypeRef?: string
  public returnType?: TypeDef<TypeDefKind>
  public alias: string | undefined
  public arguments: DaggerArguments = {}

  private signature: ts.Signature
  private symbol: ts.Symbol

  constructor(
    private readonly node: ts.MethodDeclaration,
    private readonly ast: AST,
  ) {
    super(node)

    this.symbol = this.ast.getSymbolOrThrow(node.name)
    this.signature = this.ast.getSignatureFromFunctionOrThrow(node)
    this.name = this.node.name.getText()
    this.description = this.ast.getDocFromSymbol(this.symbol)

    for (const parameter of this.node.parameters) {
      this.arguments[parameter.name.getText()] = new DaggerArgument(
        parameter,
        this.ast,
      )
    }
    this.returnType = this.getReturnType()
    this.alias = this.getAlias()
  }

  private getReturnType(): TypeDef<TypeDefKind> | undefined {
    const type = this.signature.getReturnType()

    const typedef = this.ast.tsTypeToTypeDef(this.node, type)
    if (typedef === undefined || !isTypeDefResolved(typedef)) {
      this._returnTypeRef = this.ast.typeToStringType(type)
    }

    return typedef
  }

  private getAlias(): string | undefined {
    const alias = this.ast.getDecoratorArgument<string>(
      this.node,
      FUNCTION_DECORATOR,
      "string",
    )
    if (!alias) {
      return
    }

    return JSON.parse(alias.replace(/'/g, '"'))
  }

  public getArgsOrder(): string[] {
    return Object.keys(this.arguments)
  }

  public getReferences(): string[] {
    const references: string[] = []

    if (
      this._returnTypeRef &&
      (this.returnType === undefined || !isTypeDefResolved(this.returnType))
    ) {
      references.push(this._returnTypeRef)
    }

    for (const argument of Object.values(this.arguments)) {
      const reference = argument.getReference()
      if (reference) {
        references.push(reference)
      }
    }

    return references
  }

  public propagateReferences(references: References) {
    for (const argument of Object.values(this.arguments)) {
      argument.propagateReferences(references)
    }

    if (!this._returnTypeRef) {
      return
    }

    if (this.returnType && isTypeDefResolved(this.returnType)) {
      return
    }

    const typeDef = references[this._returnTypeRef]
    if (!typeDef) {
      throw new Error(
        `could not find type reference for ${this._returnTypeRef} at ${AST.getNodePosition(this.node)}.`,
      )
    }

    this.returnType = resolveTypeDef(this.returnType, typeDef)
  }

  public toJSON() {
    return {
      name: this.name,
      description: this.description,
      alias: this.alias,
      arguments: this.arguments,
      returnType: this.returnType,
    }
  }
}
