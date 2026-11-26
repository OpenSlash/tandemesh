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

#include "Rect.h"

namespace rnoh {
namespace drawing {
Rect::Rect() : Rect(0, 0, 0, 0) {}

Rect::Rect(float left, float top, float right, float bottom)
    : rect_(UniqueNativeRect(OH_Drawing_RectCreate(left, top, right, bottom), &OH_Drawing_RectDestroy)) {}

Rect::Rect(Rect const &other) : Rect() { *this = other; }
Rect &Rect::operator=(Rect const &other) {
    OH_Drawing_RectCopy(other.get(), get());
    return *this;
}

bool Rect::Intersect(const Rect &other) { return OH_Drawing_RectIntersect(get(), other.get()); }

bool Rect::Join(const Rect &other) { return OH_Drawing_RectJoin(get(), other.get()); }

void Rect::SetLeft(float left) { OH_Drawing_RectSetLeft(get(), left); }

void Rect::SetTop(float top) { OH_Drawing_RectSetTop(get(), top); }

void Rect::SetRight(float right) { OH_Drawing_RectSetRight(get(), right); }

void Rect::SetBottom(float bottom) { OH_Drawing_RectSetBottom(get(), bottom); }

float Rect::GetLeft() { return OH_Drawing_RectGetLeft(get()); }

float Rect::GetTop() { return OH_Drawing_RectGetTop(get()); }

float Rect::GetRight() { return OH_Drawing_RectGetRight(get()); }

float Rect::GetBottom() { return OH_Drawing_RectGetBottom(get()); }

float Rect::GetHeight() { return OH_Drawing_RectGetHeight(get()); }

float Rect::GetWidth() { return OH_Drawing_RectGetWidth(get()); }

} // namespace drawing
} // namespace rnoh