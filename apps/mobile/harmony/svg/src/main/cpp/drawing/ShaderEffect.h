/**
 * MIT License
 *
 * Copyright (C) 2025 Huawei Device Co., Ltd.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

#pragma once

#include <memory>
#include <native_drawing/drawing_shader_effect.h>
#include <native_drawing/drawing_point.h>
#include <native_drawing/drawing_matrix.h>
#include <native_drawing/drawing_types.h>

namespace rnoh::drawing {

class ShaderEffect final {

    using UniqueNativeShaderEffect =
        std::unique_ptr<OH_Drawing_ShaderEffect, decltype(&OH_Drawing_ShaderEffectDestroy)>;

public:
    ShaderEffect();

    ShaderEffect(ShaderEffect const &other) = delete;
    ShaderEffect &operator=(ShaderEffect const &other) = delete;

    ShaderEffect(ShaderEffect &&other) = default;
    ShaderEffect &operator=(ShaderEffect &&other) = default;

    void ShaderEffectCreateLinearGradient(const OH_Drawing_Point2D *startPt, const OH_Drawing_Point2D *endPt,
                                          const uint32_t *colors, const float *pos, uint32_t size,
                                          OH_Drawing_TileMode mode, const OH_Drawing_Matrix *mat);

    void ShaderEffectCreateRadialGradient(const OH_Drawing_Point2D *startPt, float startRadius,
                                          const OH_Drawing_Point2D *endPt, float endRadius, const uint32_t *colors,
                                          const float *pos, uint32_t size, OH_Drawing_TileMode mode,
                                          const OH_Drawing_Matrix *mat);

    void ShaderEffectCreateImageShader(OH_Drawing_Image *image, OH_Drawing_TileMode tileX, OH_Drawing_TileMode tileY,
                                       const OH_Drawing_SamplingOptions *opt, const OH_Drawing_Matrix *mat);

    ~ShaderEffect() noexcept = default;

    OH_Drawing_ShaderEffect *get() const { return shaderEffect_.get(); }

private:
    UniqueNativeShaderEffect shaderEffect_;
};

} // namespace rnoh::drawing