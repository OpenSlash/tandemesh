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

#include "SvgMarkerPositionUtils.h"
#include "drawing/Path.h"
#include <vector>
#include <string>

namespace rnoh {
namespace svg {
class PathParserUtils {

public:
    float mScale;
    std::vector<PathElement> elements;

    drawing::Path parse(std::string d);

private:
    int i;
    int l;
    std::string s;
    drawing::Path mPath;
    float mPenX, mPenY, mPenDownX, mPenDownY, mPivotX, mPivotY;
    bool mPenDown;

    void parse_list_separator();

    void skip_digits();

    void close();

    void setPenDown();

    void moveTo(float x, float y);

    void move(float x, float y);

    void lineTo(float x, float y);

    void line(float x, float y);

    void cubicTo(float c1x, float c1y, float c2x, float c2y, float ex, float ey);

    void curveTo(float c1x, float c1y, float c2x, float c2y, float ex, float ey);

    void curve(float c1x, float c1y, float c2x, float c2y, float ex, float ey);

    void smoothCurveTo(float c1x, float c1y, float ex, float ey);

    void smoothCurve(float c1x, float c1y, float ex, float ey);

    void quadraticBezierCurveTo(float c1x, float c1y, float c2x, float c2y);

    void quadraticBezierCurve(float c1x, float c1y, float c2x, float c2y);

    void smoothQuadraticBezierCurveTo(float c1x, float c1y);

    void smoothQuadraticBezierCurve(float c1x, float c1y);

    void arcToBezier(float cx, float cy, float rx, float ry, float sa, float ea, bool clockwise, float rad);

    void arcTo(float rx, float ry, float rotation, bool outer, bool clockwise, float x, float y);

    void arc(float rx, float ry, float rotation, bool outer, bool clockwise, float x, float y);

    double roundVal(double val);

    void skip_spaces();

    bool is_cmd(char c);

    bool is_number_start(char c);

    bool is_absolute(char c);

    bool parse_flag();

    float parse_number();

    float parse_list_number();
};
} // namespace svg
} // namespace rnoh
