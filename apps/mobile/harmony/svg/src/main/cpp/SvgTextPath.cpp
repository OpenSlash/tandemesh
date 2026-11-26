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

#include "SvgTextPath.h"
#include "SvgTSpan.h"

namespace rnoh {
namespace svg {

void SvgTextPath::OnDraw(OH_Drawing_Canvas *canvas) {
    if (!glyphCtx_) {
        InitGlyph(canvas, scale_);
    }
    for (auto const &child : children_) {
        if (auto tSpan = std::dynamic_pointer_cast<SvgTSpan>(child)) {
            tSpan->SetGlyphContext(glyphCtx_);
            tSpan->SetTextPathRef(std::dynamic_pointer_cast<SvgTextPath>(shared_from_this()));
        }
    }
}

drawing::Path SvgTextPath::getTextPath() {
    auto node = context_->GetSvgNodeById(href_);
    if (node) {
        return node->AsPath();
    }
    DLOG(INFO) << "TEXT_PATH ref node missing, href = " << href_;
    return drawing::Path();
}

} // namespace svg
} // namespace rnoh