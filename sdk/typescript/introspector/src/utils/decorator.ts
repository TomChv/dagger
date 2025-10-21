export type DaggerDecorators =
  | "object"
  | "func"
  | "argument"
  | "enumType"
  | "field"

export const OBJECT_DECORATOR = "object" as DaggerDecorators
export const FUNCTION_DECORATOR = "func" as DaggerDecorators
export const FIELD_DECORATOR = "field" as DaggerDecorators
export const ARGUMENT_DECORATOR = "argument" as DaggerDecorators
export const ENUM_DECORATOR = "enumType" as DaggerDecorators

export type ArgumentOptions = {
  /**
   * The contextual value to use for the argument.
   *
   * This should only be used for Directory/File or GitRepository/GitRef types.
   *
   * An absolute path would be related to the context source directory (the git repo root or the module source root).
   * A relative path would be relative to the module source root.
   */
  defaultPath?: string

  /**
   * Patterns to ignore when loading the contextual argument value.
   *
   * This should only be used for Directory types.
   */
  ignore?: string[]
}
