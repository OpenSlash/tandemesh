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

class SvgRect : public SvgGraphic {
public:
    using Float = facebook::react::Float;
    SvgRect() = default;
    ~SvgRect() override = default;

    void SetX(const std::string &x) { rectAttribute_.x = SvgAttributesParser::ParseDimension(x, true); }
    void SetY(const std::string &y) { rectAttribute_.y = SvgAttributesParser::ParseDimension(y, true); }
    void SetRX(const std::string &rx) { rectAttribute_.rx = SvgAttributesParser::ParseDimension(rx, true); }
    void SetRY(const std::string &ry) { rectAttribute_.ry = SvgAttributesParser::ParseDimension(ry, true); }
    void SetWidth(const std::string &width) { rectAttribute_.width = SvgAttributesParser::ParseDimension(width, true); }
    void SetHeight(const std::string &height) { rectAttribute_.height = SvgAttributesParser::ParseDimension(height, true); }
    drawing::Path AsPath() override;
    Rect AsBounds() override;
private:
    SvgRectAttribute rectAttribute_;
};

} // namespace svg
} // namespace rnoh