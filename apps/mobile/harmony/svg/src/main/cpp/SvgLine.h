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
#include "utils/SvgAttributesParser.h"

namespace rnoh {
namespace svg {

class SvgLine : public SvgGraphic {
public:
    using Float = facebook::react::Float;
    SvgLine() = default;
    ~SvgLine() override = default;

    drawing::Path AsPath() override;
    Rect AsBounds() override;
    void SetX1(const std::string &x1) { lineAttribute_.x1 = SvgAttributesParser::ParseDimension(x1, true); }
    void SetY1(const std::string &y1) { lineAttribute_.y1 = SvgAttributesParser::ParseDimension(y1, true); }
    void SetX2(const std::string &x2) { lineAttribute_.x2 = SvgAttributesParser::ParseDimension(x2, true); }
    void SetY2(const std::string &y2) { lineAttribute_.y2 = SvgAttributesParser::ParseDimension(y2, true); }

private:
    SvgLineAttribute lineAttribute_;
};

} // namespace svg
} // namespace rnoh