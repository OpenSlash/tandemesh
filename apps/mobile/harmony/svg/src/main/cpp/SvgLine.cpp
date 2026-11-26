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

#include "SvgLine.h"

namespace rnoh {
namespace svg {

drawing::Path SvgLine::AsPath() {
    drawing::Path path;
    double x1 = relativeOnWidth(lineAttribute_.x1);
    double y1 = relativeOnHeight(lineAttribute_.y1);
    double x2 = relativeOnWidth(lineAttribute_.x2);
    double y2 = relativeOnHeight(lineAttribute_.y2);

    path.MoveTo(x1, y1);
    path.LineTo(x2, y2);

    elements_ = {PathElement(ElementType::kCGPathElementMoveToPoint, {Point(x1, y1)}),
                 PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(x2, y2)})};
    return path;
};

Rect SvgLine::AsBounds() {
    double x1 = relativeOnWidth(lineAttribute_.x1);
    double y1 = relativeOnHeight(lineAttribute_.y1);
    double x2 = relativeOnWidth(lineAttribute_.x2);
    double y2 = relativeOnHeight(lineAttribute_.y2);
    double x = std::min(x1, x2);
    double y = std::min(y1, y2);
    double w = std::abs(x2 - x1);
    double h = std::abs(y2 - y1);
    return Rect(x, y, w, h);
}

} // namespace svg
} // namespace rnoh