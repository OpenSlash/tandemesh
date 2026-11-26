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

#include "SvgCircle.h"

namespace rnoh {
namespace svg {

drawing::Path SvgCircle::AsPath() {
    drawing::Path path;
    double x = relativeOnWidth(circleAttribute_.cx);
    double y = relativeOnHeight(circleAttribute_.cy);
    double r = relativeOnOther(circleAttribute_.r);
    
    path.AddCircle(x, y, r, PATH_DIRECTION_CW);
    
    elements_ = {PathElement(ElementType::kCGPathElementMoveToPoint, {Point(x, y - r)}),
                       PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(x, y - r), Point(x + r, y)}),
                       PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(x + r, y), Point(x, y + r)}),
                       PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(x, y + r), Point(x - r, y)}),
                       PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(x - r, y), Point(x, y - r)})};
    return path;
}

Rect SvgCircle::AsBounds() {
    double x = relativeOnWidth(circleAttribute_.cx);
    double y = relativeOnHeight(circleAttribute_.cy);
    double r = relativeOnOther(circleAttribute_.r);
    return Rect(x - r, y - r, 2 * r, 2 * r);
}

} // namespace svg
} // namespace rnoh
