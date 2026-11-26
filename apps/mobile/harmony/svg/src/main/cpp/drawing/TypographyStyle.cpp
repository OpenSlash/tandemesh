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

#include "TypographyStyle.h"
#include <native_drawing/drawing_text_typography.h>

namespace rnoh::drawing {

TypographyStyle::TypographyStyle()
    : typographyStyle_(UniqueTypographyStyle(OH_Drawing_CreateTypographyStyle(), OH_Drawing_DestroyTypographyStyle)) {}

TypographyStyle::~TypographyStyle() noexcept {}

void TypographyStyle::SetTextStyle(TextStyle textStyle) {
    textStyle_ = std::move(textStyle.textStyle_);
    OH_Drawing_SetTypographyTextStyle(typographyStyle_.get(), textStyle_.get());
}

void TypographyStyle::Update(const std::shared_ptr<svg::FontData> &style) {
    auto *ts = typographyStyle_.get();
    OH_Drawing_SetTypographyTextFontWeight(ts, style->absoluteFontWeight);
    OH_Drawing_SetTypographyTextFontStyle(ts, static_cast<int>(style->fontStyle));
    OH_Drawing_SetTypographyTextFontFamily(ts, style->fontFamily.c_str());
    OH_Drawing_SetTypographyTextFontSize(ts, style->fontSize);
    // to be done? TypographyTextAlign
}

} // namespace rnoh::drawing
