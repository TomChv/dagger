import ts from "typescript"

/**
 * Return true if the given class declaration has the decorator @obj on
 * top of its declaration.
 * @param object
 */
export function isObject(object: ts.ClassDeclaration): boolean {
  return (
    ts
      .getDecorators(object)
      ?.find((d) => d.expression.getText() === "object") !== undefined
  )
}

/**
 * Return true if the given method has the decorator @fct on top
 * of its declaration.
 *
 * @param method The method to check
 */
export function isFunction(method: ts.MethodDeclaration): boolean {
  return (
    ts.getDecorators(method)?.find((d) => d.expression.getText() === "fct") !==
    undefined
  )
}

/**
 * Return true if the given property is public.
 *
 * This function actually in work the reverse, it checks if the property
 * isn't private nor protected.
 *
 * It returns true if the property has no modifiers since no keyword
 * has been set on the property.
 *
 * Example
 * ```
 * class Human {
 *     private age = 22 // Return false
 *     protected familyName = "Doe" // Return false
 *     name = "John" // Return true
 * }
 * ```
 *
 * @param property The property to check on.
 */
export function isPublicProperty(property: ts.PropertyDeclaration): boolean {
  const modifiers = ts.getModifiers(property)
  if (!modifiers) {
    return true
  }

  return !modifiers.some(
    (modifier) =>
      modifier.kind === ts.SyntaxKind.PrivateKeyword ||
      modifier.kind === ts.SyntaxKind.ProtectedKeyword
  )
}
