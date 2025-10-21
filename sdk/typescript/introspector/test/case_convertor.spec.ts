import assert from "assert"

import * as strcase from "../src/utils/strcase"

describe("strcase", function () {
  describe("PascalCase", function () {
    it("should convert kebab-case to pascal case", function () {
      const result = strcase.PascalCase("hello-world")
      assert.equal(result, "HelloWorld")
    })

    it("should convert snake-case to pascal case", function () {
      const result = strcase.PascalCase("hello_world")
      assert.equal(result, "HelloWorld")
    })

    it("should convert camel case to pascal case", function () {
      const result = strcase.PascalCase("helloWorld")
      assert.equal(result, "HelloWorld")
    })

    it("should convert single word to pascal case", function () {
      const result = strcase.PascalCase("hello")
      assert.equal(result, "Hello")
    })

    it("should convert empty string to empty string", function () {
      const result = strcase.PascalCase("")
      assert.equal(result, "")
    })

    it("should keep pascal case if already valid", function () {
      const result = strcase.PascalCase("HelloWorld")
      assert.equal(result, "HelloWorld")
    })

    it("should correctly handle numbers as spacers", function () {
      const result = strcase.PascalCase("hello123world")
      assert.equal(result, "Hello123World")
    })

    it("should correctly handle number as first word", function () {
      const result = strcase.PascalCase("123helloworld")
      assert.equal(result, "123Helloworld")
    })

    it("should correctly handle number as last word", function () {
      const result = strcase.PascalCase("helloWorld123")
      assert.equal(result, "HelloWorld123")
    })

    it("should correctly handle special characters as spacers", function () {
      const result = strcase.PascalCase("hello world")
      assert.equal(result, "HelloWorld")
    })

    it("should correctly handle mix cases", function () {
      const result = strcase.PascalCase("123hello-world_here")
      assert.equal(result, "123HelloWorldHere")
    })

    it("should correctly handle multiple uppercase words", function () {
      // This will still not work in module because the generated class name would be `IntrospectionJson`
      // See https://github.com/dagger/dagger/issues/7941
      const result = strcase.PascalCase("introspectionJSON")
      assert.equal(result, "IntrospectionJSON")
    })
  })
})
