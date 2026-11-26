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

#include <string>

namespace rnoh {
namespace svg {

enum class AlignmentBaseline {
    baseline,
    textBottom,
    alphabetic,
    ideographic,
    middle,
    central,
    mathematical,
    textTop,
    bottom,
    center,
    top,
    /*
        SVG implementations may support the following aliases in order to support legacy content:

        text-before-edge = text-top
        text-after-edge = text-bottom
    */
    textBeforeEdge,
    textAfterEdge,
    // SVG 1.1
    beforeEdge,
    afterEdge,
    hanging,
};

enum class TextLengthAdjust { spacing, spacingAndGlyphs };

enum class TextPathMethod { align, stretch };
enum class TextPathMidLine { sharp, smooth };
enum class TextPathSide { left, right };
enum class TextPathSpacing { Auto, Exact };

AlignmentBaseline alignmentBaselineFromStr(const std::string &str);
TextLengthAdjust textLengthAdjustFromStr(const std::string &str);

TextPathMethod textPathMethodFromStr(const std::string &str);
TextPathSide textPathSideFromStr(const std::string &str);
TextPathMidLine textPathMidLineFromStr(const std::string &str);
TextPathSpacing textPathSpacingFromStr(const std::string &str);

} // namespace svg
} // namespace rnoh