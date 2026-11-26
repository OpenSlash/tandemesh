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
#include <native_drawing/drawing_round_rect.h>
#include "Rect.h"

namespace rnoh {
namespace drawing {
class RoundRect final {
    using UniqueNativeRoundRect = std::unique_ptr<OH_Drawing_RoundRect, decltype(&OH_Drawing_RoundRectDestroy)>;

    using CornerPos = OH_Drawing_CornerPos;
    using CornerRadii = OH_Drawing_Corner_Radii;

public:
    RoundRect(Rect rect, float rx, float ry);
    RoundRect(RoundRect const &) = delete;
    RoundRect &operator=(RoundRect const &) = delete;
    RoundRect(RoundRect &&) = default;
    RoundRect &operator=(RoundRect &&) = default;
    ~RoundRect() noexcept = default;

    void SetCorner(CornerPos pos, CornerRadii radii);
    CornerRadii GetCorner(CornerPos pos);

    OH_Drawing_RoundRect *get() const { return roundRect_.get(); }

private:
    Rect rect_;
    UniqueNativeRoundRect roundRect_;
};
} // namespace drawing
} // namespace rnoh