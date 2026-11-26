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

#include <memory>
#include <native_drawing/drawing_text_typography.h>
#include "properties/Font.h"

namespace rnoh::drawing {

class TextStyle final {

    using UniqueTextStyle = std::unique_ptr<OH_Drawing_TextStyle, decltype(&OH_Drawing_DestroyTextStyle)>;

public:
    TextStyle();
    ~TextStyle() noexcept = default;

    TextStyle(const TextStyle &) = delete;
    TextStyle &operator=(const TextStyle &) = delete;

    TextStyle(TextStyle &&) = default;
    TextStyle &operator=(TextStyle &&) = default;

    void SetForegroundPen(OH_Drawing_Pen *pen) { OH_Drawing_SetTextStyleForegroundPen(textStyle_.get(), pen); }

    void SetForegroundBrush(OH_Drawing_Brush *brush) {
        OH_Drawing_SetTextStyleForegroundBrush(textStyle_.get(), brush);
    }

    void Update(const std::shared_ptr<svg::FontData> &style);
// private:
    UniqueTextStyle textStyle_;
    friend class TypographyStyle;
};

} // namespace rnoh::drawing