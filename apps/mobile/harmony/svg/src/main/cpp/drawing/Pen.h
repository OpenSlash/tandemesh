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

#include "ShaderEffect.h"
#include <memory>
#include <native_drawing/drawing_pen.h>
#include <native_drawing/drawing_shader_effect.h>

namespace rnoh::drawing {

class Pen final {

    using UniqueNativePen = std::unique_ptr<OH_Drawing_Pen, decltype(&OH_Drawing_PenDestroy)>;

public:
    using LineCapStyle = OH_Drawing_PenLineCapStyle;
    using LineJoinStyle = OH_Drawing_PenLineJoinStyle;

    Pen();

    Pen(Pen const &);
    Pen &operator=(Pen const &);

    Pen(Pen &&) = default;
    Pen &operator=(Pen &&) = default;

    ~Pen() noexcept = default;

    void SetAntiAlias(bool antiAlias);
    void SetColor(uint32_t color);
    void SetAlpha(uint8_t alpha);
    void SetWidth(double width);
    void SetMiterLimit(float miterLimit);
    void SetLineCap(LineCapStyle lineCap);
    void SetLineJoin(LineJoinStyle lineJoin);
    void SetLinearShaderEffect(const OH_Drawing_Point2D *startPt, const OH_Drawing_Point2D *endPt,
                               const uint32_t *colors, const float *pos, uint32_t size, OH_Drawing_TileMode mode,
                               const OH_Drawing_Matrix *mat);
    void SetRadialShaderEffect(const OH_Drawing_Point2D *startPt, float startRadius, const OH_Drawing_Point2D *endPt,
                               float endRadius, const uint32_t *colors, const float *pos, uint32_t size,
                               OH_Drawing_TileMode mode, const OH_Drawing_Matrix *mat);
    void SetImageShaderEffect(OH_Drawing_Image *image, OH_Drawing_TileMode tileX, OH_Drawing_TileMode tileY,
                              const OH_Drawing_SamplingOptions *opt, const OH_Drawing_Matrix *mat);

    void Reset();

    OH_Drawing_Pen *get() const { return pen_.get(); }

private:
    UniqueNativePen pen_;
    ShaderEffect penShaderEffect_;
};

} // namespace rnoh::drawing