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

#include "SvgGraphic.h"
#include <native_drawing/drawing_point.h>
#include "utils/SvgAttributesParser.h"

namespace rnoh {
namespace svg {

class SvgGradient : public SvgNode {
public:
    using Float = facebook::react::Float;
    explicit SvgGradient(GradientType gradientType);
    ~SvgGradient() override = default;
    
    void OnDraw(OH_Drawing_Canvas *canvas) override;

    void SetAttrX1(const std::string& x1);
    void SetAttrY1(const std::string& y1);
    void SetAttrX2(const std::string& x2);
    void SetAttrY2(const std::string& y2);

    void SetAttrFx(const std::string& fx);
    void SetAttrFy(const std::string& fy);
    void SetAttrCx(const std::string& cx);
    void SetAttrCy(const std::string& cy);
    void SetAttrRx(const std::string& rx);
    void SetAttrRy(const std::string& ry);

    void SetAttrGradient(std::vector<Float> gradient);
    void SetAttrGradientUnits(int gradientUnits);
    void SetAttrGradientTransforms(std::vector<Float> gradientTransforms);

    const Gradient& GetGradient() const;

private:
    SvgGradientAttribute gradientAttr_;
    double scale_ = 3.25;
    DimensionUnit defaultUnit_ = DimensionUnit::VP;
};

} // namespace svg
} // namespace rnoh
