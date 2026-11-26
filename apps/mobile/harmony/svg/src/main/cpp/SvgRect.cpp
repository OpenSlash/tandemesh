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

#include "SvgRect.h"
#include "drawing/Rect.h"

namespace rnoh {
namespace svg {

drawing::Path SvgRect::AsPath() {
    drawing::Path path;
    double x = relativeOnWidth(rectAttribute_.x);
    double y = relativeOnHeight(rectAttribute_.y);
    double width = relativeOnWidth(rectAttribute_.width);
    double height = relativeOnHeight(rectAttribute_.height);

    if (rectAttribute_.rx != Dimension(-1.0, DimensionUnit::PX) ||
        rectAttribute_.ry != Dimension(-1.0, DimensionUnit::PX)) {
        double rx;
        double ry;
        if (rectAttribute_.rx == Dimension(-1.0, DimensionUnit::PX)) {
            ry = relativeOnHeight(rectAttribute_.ry);
            rx = ry;
        } else if (rectAttribute_.ry == Dimension(-1.0, DimensionUnit::PX)) {
            rx = relativeOnWidth(rectAttribute_.rx);
            ry = rx;
        } else {
            rx = relativeOnWidth(rectAttribute_.rx);
            ry = relativeOnHeight(rectAttribute_.ry);
        }
        
        if (rx > width / 2.0) {
            rx = width / 2.0;
        }

        if (ry > height / 2.0) {
            ry = height / 2.0;
        }
        drawing::Rect rect(x, y, x + width, y + height);
        drawing::RoundRect roundRect(std::move(rect), rx, ry);
        path.AddRoundRect(roundRect, PATH_DIRECTION_CW);
    } else {
        path.AddRect(x, y, x + width, y + height, PATH_DIRECTION_CW);
    }

    elements_ = {PathElement(ElementType::kCGPathElementMoveToPoint, {Point(x, y)}),
                 PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(x + width, y)}),
                 PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(x + width, y + height)}),
                 PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(x, y + height)}),
                 PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(x, y)})};
    return path;
};

Rect SvgRect::AsBounds() {
    double x = relativeOnWidth(rectAttribute_.x);
    double y = relativeOnHeight(rectAttribute_.y);
    double width = relativeOnWidth(rectAttribute_.width);
    double height = relativeOnHeight(rectAttribute_.height);
    return Rect(x, y, width, height);
}

} // namespace svg
} // namespace rnoh
