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

#include <memory>
#include "SvgText.h"
#include "drawing/TextStyle.h"
#include "drawing/TypographyStyle.h"
#include "SvgTSpan.h"

namespace rnoh {
namespace svg {

void SvgText::GlyphTraversal(OH_Drawing_Canvas *canvas) {
    if (!glyphCtx_) {
        InitGlyph(canvas, scale_);
    }
    for (auto const &child : children_) {
        if (auto span = std::dynamic_pointer_cast<TextBase>(child)) {
            span->SetGlyphContext(glyphCtx_);
            if (!span->hasAlignmentBaseline()) {
                span->setAlignmentBaseline(align_);
            }
            if (!span->hasBaselineShift()) {
                span->setBaselineShift(baselineShift_);
            }
        }
    }
}

void SvgText::OnDraw(OH_Drawing_Canvas *canvas) {
    GlyphTraversal(canvas);
    glyphCtx_->pushContext(true, shared_from_this(), x_, y_, dx_, dy_, rotate_);
}

drawing::Path SvgText::AsPath() {
    drawing::Path resultPath;
    for (auto const &child : children_) {
        if (auto span = std::dynamic_pointer_cast<SvgTSpan>(child)) {
            std::string text = span->content_;
        }
    }
    return resultPath;
}
} // namespace svg
} // namespace rnoh