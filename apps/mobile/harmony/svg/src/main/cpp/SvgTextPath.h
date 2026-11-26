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

#include "SvgGroup.h"
#include "SvgText.h"
#include <native_drawing/drawing_text_typography.h>
#include "utils/GlyphContext.h"
#include "properties/Font.h"
#include "TextBase.h"

namespace rnoh {
namespace svg {

class SvgTextPath : public SvgGroup, public TextBase {
public:
    SvgTextPath() {
        hrefFill_ = true;
        hrefRender_ = true;
        passStyle_ = true;
        inheritStyle_ = true;
        drawTraversed_ = true;
    }
    ~SvgTextPath() override = default;

    void OnDraw(OH_Drawing_Canvas *canvas) override;

    void SetContext(std::shared_ptr<GlyphContext> context) { glyphCtx_ = context; }

    drawing::Path getTextPath();

    TextPathMidLine getMidLine() const { return midLine_; }

    TextPathSide getSide() const { return side_; }

    double getStartOffset() const { return startOffset_; }

    TextPathSide side_{TextPathSide::left};
    TextPathMidLine midLine_{TextPathMidLine::sharp};
    TextPathMethod method_{TextPathMethod::align};
    TextPathSpacing spacing_{TextPathSpacing::Exact};
    double startOffset_;
    std::string href_;
};

} // namespace svg
} // namespace rnoh