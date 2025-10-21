import { func, object } from "@dagger.io/dagger"
import { ImageLayerCompression } from "@dagger.io/dagger"

@object()
export class CoreEnums {
  @func()
  toImageLayerCompression(compression: string): ImageLayerCompression {
    return compression as ImageLayerCompression
  }

  @func()
  fromImageLayerCompression(compression: ImageLayerCompression): string {
    return compression
  }
}
