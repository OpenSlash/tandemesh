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
#include "properties/ViewBox.h"

namespace rnoh {
namespace svg {

class SvgSymbol : public SvgGraphic {
public:
    SvgSymbol() : SvgGraphic() { InitSymbolFlag(); }
    ~SvgSymbol() override = default;

    void InitSymbolFlag() {
        hrefFill_ = false;
        hrefRender_ = true;
        inheritStyle_ = false;
        drawTraversed_ = false;
    }
    float mMinX = 0.0f;
    float mMinY = 0.0f;
    float mVbWidth = 0.0f;
    float mVbHeight = 0.0f;
    std::string mAlign;
    int mMeetOrSlice = -1;
    void InitDefsFlag();
    void drawSymbol(OH_Drawing_Canvas *canvas, float width, float height);
};

} // namespace svg
} // namespace rnoh
