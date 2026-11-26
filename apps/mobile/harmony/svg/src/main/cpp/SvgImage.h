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

#include <string>
#include "SvgGraphic.h"
#include <rawfile/raw_file_manager.h>

namespace rnoh {
namespace svg {

class SvgImage : public svg::SvgGraphic {
public:
    SvgImage() = default;
    ~SvgImage() override = default;

    using SvgNode::relativeOnWidth;
    using SvgNode::relativeOnHeight;

    void SetX(const std::string &x) { imageAttribute_.x = SvgAttributesParser::ParseDimension(x, true); }

    void SetY(const std::string &y) { imageAttribute_.y = SvgAttributesParser::ParseDimension(y, true); }

    void SetHeight(const std::string &height) {
        imageAttribute_.height = SvgAttributesParser::ParseDimension(height, true);
    }

    void SetWidth(const std::string &width) {
        imageAttribute_.width = SvgAttributesParser::ParseDimension(width, true);
    }

    void SetAlign(const std::string &align) { imageAttribute_.align = align; }

    void SetMeetOrSlice(const int &meetOrSlice) { imageAttribute_.meetOrSlice = meetOrSlice; }

    void SetImageSource(const facebook::react::ImageSource &src) { imageAttribute_.src = src; }

    void setNativeResourceManager(const NativeResourceManager *mgr) { mgr_ = mgr; }

    void OnDraw(OH_Drawing_Canvas *canvas) override;

    Rect AsBounds() override;

private:
    SvgImageAttribute imageAttribute_;
    const NativeResourceManager *mgr_;
};

} // namespace svg
} // namespace rnoh
