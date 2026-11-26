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

#include "Rect.h"
#include <memory>
#include <optional>
#include <vector>
#include <native_drawing/drawing_matrix.h>

namespace rnoh::drawing {

class Matrix final {
    using UniqueNativeMatrix = std::unique_ptr<OH_Drawing_Matrix, decltype(&OH_Drawing_MatrixDestroy)>;
    using Point = OH_Drawing_Point2D;

public:
    Matrix();

    Matrix(Matrix const &);
    Matrix &operator=(Matrix const &);

    Matrix(Matrix &&) = default;
    Matrix &operator=(Matrix &&) = default;

    ~Matrix() noexcept = default;

    static Matrix CreateRotation(float deg, float x, float y);
    static Matrix CreateScale(float sx, float sy, float px, float py);
    static Matrix CreateTranslation(float dx, float dy);
    void PreRotate(float degree, float px, float py);
    void PreScale(float sx, float sy, float px, float py);
    void PreTranslate(float dx, float dy);
    void PostRotate(float degree, float px, float py);
    void PostScale(float sx, float sy, float px, float py);
    void PostTranslate(float dx, float dy);
    void Rotate(float degree, float px, float py);
    void Translate(float dx, float dy);
    void Scale(float sx, float sy, float px, float py);

    Matrix Concat(const Matrix &other) const;
    std::optional<Matrix> Invert() const;

    float GetValue(int index) const;

    void SetMatrix(float scaleX, float skewX, float transX, float skewY, float scaleY, float transY, float persp0,
                   float persp1, float persp2);

    Point MapPoint(const Point &point) const;
    std::vector<Point> MapPoints(const std::vector<Point> &points) const;

    drawing::Rect MapRect(const drawing::Rect &rect) const;

    void Reset();

    OH_Drawing_Matrix *get() const { return matrix_.get(); }

private:
    Matrix(OH_Drawing_Matrix *rawMatrix) : matrix_(UniqueNativeMatrix(rawMatrix, &OH_Drawing_MatrixDestroy)) {}

    UniqueNativeMatrix matrix_;
};

} // namespace rnoh::drawing