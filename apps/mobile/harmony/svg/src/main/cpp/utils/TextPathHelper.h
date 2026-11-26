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

#include "drawing/Matrix.h"
#include "SvgTextPath.h"

namespace rnoh {
namespace svg {

class TextPathHelper {
    public:
    TextPathHelper(const std::shared_ptr<svg::SvgTextPath>& textPath, svg::TextAnchor anchor);

    double GetStartOffset() const {
        return absoluteStartOffset_;
    }

    int GetSide() const {
        return side_;
    }

    bool Valid() const {
        return pathLength_ > 0.0;
    }

    void SetScaleSpacingAndGlyphs(double value) {
        scaleDirection_ = value * side_;
    }

    struct Params {
        double charWidth = 0.0;
        double startPoint = 0.0;
        double y = 0.0;
        double dy = 0.0;
    };
    /**
     * @brief Set transform matrix to paint the character at the given position on the path.
     *
     * @return true if this character can be skipped
     */
    bool GetMatrixOnPath(const Params& p, drawing::Matrix &mat);

private:
    drawing::Path path_ = drawing::Path();
    int side_ = 0;
    double pathLength_ = 0.0;
    double startOfRendering_ = 0.0;
    double endOfRendering_ = 0.0;
    double absoluteStartOffset_ = 0.0;
    double scaleDirection_ = 1.0;
    bool sharpMidLine_ = false;
    bool isClosed_ = false;
};

} // namespace svg
} // namespace rnoh