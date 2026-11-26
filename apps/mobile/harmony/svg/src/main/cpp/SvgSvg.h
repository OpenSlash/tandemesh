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

#include <native_drawing/drawing_canvas.h>
#include <native_drawing/drawing_path.h>
#include "SvgGroup.h"
#include "utils/SvgAttributesParser.h"

namespace rnoh {
namespace svg {

class SvgSvg : public SvgGroup {
public:
    SvgSvg();
    ~SvgSvg() override = default;

    drawing::Path AsPath() override;
    
    std::shared_ptr<SvgNode> HitTest(double x, double y) override;

    Size GetSize() const;
    
    Rect GetViewBox() const;
    
    void Draw(OH_Drawing_Canvas *canvas) override;

    void SetVbX(const double &vbX) { svgAttribute_.vbX = Dimension(vbX, DimensionUnit::VP); }
    void SetVbY(const double &vbY) { svgAttribute_.vbY = Dimension(vbY, DimensionUnit::VP); }
    void SetVbWidth(const double &vbWidth) { svgAttribute_.vbWidth = Dimension(vbWidth, DimensionUnit::VP); }
    void SetVbHeight(const double &vbHeight) { svgAttribute_.vbHeight = Dimension(vbHeight, DimensionUnit::VP); }
    void SetX(const double &x) { svgAttribute_.x = Dimension(x, DimensionUnit::VP); }
    void SetY(const double &y) { svgAttribute_.y = Dimension(y, DimensionUnit::VP); }
    void SetWidth(const std::string &width) { svgAttribute_.width = SvgAttributesParser::ParseDimension(width, true); }
    void SetHeight(const std::string &height) { svgAttribute_.height = SvgAttributesParser::ParseDimension(height, true); }
    void SetAlign(const std::string &align) { svgAttribute_.align = align; }
    void SetMeetOrSlice(const int &meetOrSlice) { svgAttribute_.meetOrSlice = meetOrSlice; }

private:
    SvgAttribute svgAttribute_;
    
    float width_ = 0;
    
    float height_ = 0;
    
    drawing::Matrix FitCanvas(OH_Drawing_Canvas *canvas);
};

} // namespace svg
} // namespace rnoh
