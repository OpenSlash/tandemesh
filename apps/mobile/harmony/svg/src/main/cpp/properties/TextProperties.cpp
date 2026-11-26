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

#include "TextProperties.h"

namespace rnoh {
namespace svg {

TextPathMethod textPathMethodFromStr(const std::string &str) {
    if (str == "stretch") {
        return TextPathMethod::stretch;
    } else {
        return TextPathMethod::align;
    }
}

TextPathSide textPathSideFromStr(const std::string &str) {
    if (str == "right") {
        return TextPathSide::right;
    } else {
        return TextPathSide::left;
    }
}

TextPathMidLine textPathMidLineFromStr(const std::string &str) {
    if (str == "sharp") {
        return TextPathMidLine::sharp;
    } else {
        return TextPathMidLine::smooth;
    }
}

TextPathSpacing textPathSpacingFromStr(const std::string &str) {
    if (str == "auto") {
        return TextPathSpacing::Auto;
    } else {
        return TextPathSpacing::Exact;
    }
}

TextLengthAdjust textLengthAdjustFromStr(const std::string &str) {
    if (str == "spacing") {
        return TextLengthAdjust::spacing;
    } else {
        return TextLengthAdjust::spacingAndGlyphs;
    }
}

AlignmentBaseline alignmentBaselineFromStr(const std::string &str) {
    if (str == "baseline") {
        return AlignmentBaseline::baseline;
    } else if (str == "text-bottom") {
        return AlignmentBaseline::textBottom;
    } else if (str == "alphabetic") {
        return AlignmentBaseline::alphabetic;
    } else if (str == "ideographic") {
        return AlignmentBaseline::ideographic;
    } else if (str == "middle") {
        return AlignmentBaseline::middle;
    } else if (str == "central") {
        return AlignmentBaseline::central;
    } else if (str == "mathematical") {
        return AlignmentBaseline::mathematical;
    } else if (str == "text-top") {
        return AlignmentBaseline::textTop;
    } else if (str == "bottom") {
        return AlignmentBaseline::bottom;
    } else if (str == "center") {
        return AlignmentBaseline::center;
    } else if (str == "top") {
        return AlignmentBaseline::top;
    } else if (str == "text-before-edge") {
        return AlignmentBaseline::textBeforeEdge;
    } else if (str == "text-after-edge") {
        return AlignmentBaseline::textAfterEdge;
    } else if (str == "before-edge") {
        return AlignmentBaseline::beforeEdge;
    } else if (str == "after-edge") {
        return AlignmentBaseline::afterEdge;
    } else if (str == "hanging") {
        return AlignmentBaseline::hanging;
    } else {
        return AlignmentBaseline::baseline;
    }
}

} // namespace svg
} // namespace rnoh