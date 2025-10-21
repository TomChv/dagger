import { func, object } from "@dagger.io/dagger"
import type { Platform } from "@dagger.io/dagger"

@object()
export class Scalar {
  @func()
  fromPlatform(platform: Platform): string {
    return platform as string
  }

  @func()
  fromPlatforms(platforms: Platform[]): string[] {
    return platforms.map((p) => p as string)
  }
}
