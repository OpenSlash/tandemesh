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

#include "SvgSymbol.h"

namespace rnoh {
namespace svg {

void SvgSymbol::drawSymbol(OH_Drawing_Canvas *canvas, float width, float height) {
    if (!mAlign.empty()) {
        Rect vbRect(mMinX * scale_, mMinY * scale_, mMinX + mVbWidth * scale_, mMinY + mVbHeight * scale_);
        Rect eRect(0, 0, width * scale_, height * scale_);
        if (vbRect.IsValid()) {
			if (mMeetOrSlice == 1) {
                drawing::Rect clipRect(0.0f, 0.0f, width * scale_, height * scale_);
                OH_Drawing_CanvasClipRect(canvas, clipRect.get(), OH_Drawing_CanvasClipOp::INTERSECT, true);
            }
            drawing::Matrix viewBoxMatrix = ViewBox::getTransform(vbRect, eRect, mAlign, mMeetOrSlice);
            OH_Drawing_CanvasConcatMatrix(canvas, viewBoxMatrix.get());
        }
    }
    Draw(canvas);
}

} // namespace svg
} // namespace rnoh