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
#include <native_drawing/drawing_rect.h>

namespace rnoh {
namespace drawing {
class Rect final {
    using UniqueNativeRect = std::unique_ptr<OH_Drawing_Rect, decltype(&OH_Drawing_RectDestroy)>;

public:
    Rect();
    Rect(float left, float top, float right, float bottom);
    Rect(Rect const &);
    Rect &operator=(Rect const &);
    Rect(Rect &&) = default;
    Rect &operator=(Rect &&) = default;
    ~Rect() noexcept = default;

    bool Intersect(const Rect &other);

    bool Join(const Rect &other);

    void SetLeft(float left);

    void SetTop(float top);

    void SetRight(float right);

    void SetBottom(float bottom);

    float GetLeft();

    float GetTop();

    float GetRight();

    float GetBottom();

    float GetHeight();

    float GetWidth();

    OH_Drawing_Rect *get() const { return rect_.get(); }

private:
    UniqueNativeRect rect_;
};
} // namespace drawing
} // namespace rnoh