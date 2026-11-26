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

#include "drawing/Rect.h"
#include "properties/Dimension.h"
#include "SvgQuote.h"

namespace rnoh {
namespace svg {

class SvgMask : public SvgQuote {
public:
    SvgMask() = default;
    ~SvgMask() override = default;

    void setMaskX(const std::string &x) {
        maskAttribute_.x = StringUtils::StringToDimensionWithUnit(x, defaultDimensionUnit_);
    }

    void setMaskY(const std::string &y) {
        maskAttribute_.y = StringUtils::StringToDimensionWithUnit(y, defaultDimensionUnit_);
    }

    void setMaskHeight(const std::string &height) {
        maskAttribute_.height = StringUtils::StringToDimensionWithUnit(height, defaultDimensionUnit_);
    }

    void setMaskWidth(const std::string &width) {
        maskAttribute_.width = StringUtils::StringToDimensionWithUnit(width, defaultDimensionUnit_);
    }

    void isDefaultMaskUnits(bool isDefaultMaskUnits) { isDefaultMaskUnits_ = isDefaultMaskUnits; }

    void isDefaultMaskContentUnits(bool isDefaultMaskContentUnits) {
        isDefaultMaskContentUnits_ = isDefaultMaskContentUnits;
    }

    void setMaskUnits(const int &maskUnits) {
        maskAttribute_.maskUnits = ToUnit(maskUnits);
        isDefaultMaskUnits(maskAttribute_.maskUnits == Unit::objectBoundingBox);
    }

    void setMaskContentUnits(const int &maskContentUnits) {
        maskAttribute_.maskContentUnits = ToUnit(maskContentUnits);
        isDefaultMaskContentUnits(maskAttribute_.maskContentUnits == Unit::userSpaceOnUse);
    }
	
    void SetHrefBounds(const Rect &hostMaskBounds) { hostMaskBounds_ = hostMaskBounds; }

    Rect GetHrefBounds() { return hostMaskBounds_; }

protected:
    void OnInitStyle() override;
    void OnDrawTraversedBefore(OH_Drawing_Canvas *canvas) override;
    void OnDrawTraversedAfter(OH_Drawing_Canvas *canvas) override;
    double ParseUnitsAttr(const Dimension &attr, double value);

private:
    bool isDefaultMaskUnits_ = true;
    bool isDefaultMaskContentUnits_ = true;
    SvgMaskAttribute maskAttribute_;

    drawing::Rect maskBounds_;
	Rect hostMaskBounds_;
    int canvasLayerCount_ = -1;
    DimensionUnit defaultDimensionUnit_ = DimensionUnit::VP;
};

} // namespace svg
} // namespace rnoh