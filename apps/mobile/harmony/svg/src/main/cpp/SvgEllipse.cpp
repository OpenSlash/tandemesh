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

#include "SvgEllipse.h"
#include "drawing/Rect.h"

namespace rnoh {
namespace svg {

drawing::Path SvgEllipse::AsPath() {
    drawing::Path path;
    double cx = relativeOnWidth(ellipseAttribute_.cx);
    double cy = relativeOnHeight(ellipseAttribute_.cy);
    double rx = relativeOnWidth(ellipseAttribute_.rx);
    double ry = relativeOnHeight(ellipseAttribute_.ry);

    drawing::Rect rect(cx - rx, cy - ry, cx + rx, cy + ry);
    path.AddOval(rect, PATH_DIRECTION_CW);

    elements_ = {PathElement(ElementType::kCGPathElementMoveToPoint, {Point(cx, cy - ry)}),
                 PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(cx, cy - ry), Point(cx + rx, cy)}),
                 PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(cx + rx, cy), Point(cx, cy + ry)}),
                 PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(cx, cy + ry), Point(cx - rx, cy)}),
                 PathElement(ElementType::kCGPathElementAddLineToPoint, {Point(cx - rx, cy), Point(cx, cy - ry)})};
    return path;
};

Rect SvgEllipse::AsBounds() {
    double cx = relativeOnWidth(ellipseAttribute_.cx);
    double cy = relativeOnHeight(ellipseAttribute_.cy);
    double rx = relativeOnWidth(ellipseAttribute_.rx);
    double ry = relativeOnHeight(ellipseAttribute_.ry);
    return Rect(cx - rx, cy - ry, 2 * rx, 2 * ry);
}

} // namespace svg
} // namespace rnoh