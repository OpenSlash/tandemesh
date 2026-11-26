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

#include "utils/StringUtils.h"
#include "utils/GlyphContext.h"
#include "properties/TextProperties.h"
#include <native_drawing/drawing_text_typography.h>
#include <optional>

namespace rnoh {
namespace svg {

class TextBase {
public:
    template <typename T> void UpdateTextProps(const std::shared_ptr<T> &props) {
        x_.clear();
        for (auto const &value : props->x) {
            x_.push_back(StringUtils::FromString(DynamicUtils::DynamicToString(value)));
        }
        y_.clear();
        for (auto const &value : props->y) {
            y_.push_back(StringUtils::FromString(DynamicUtils::DynamicToString(value)));
        }
        dx_.clear();
        for (auto const &value : props->dx) {
            dx_.push_back(StringUtils::FromString(DynamicUtils::DynamicToString(value)));
        }
        dy_.clear();
        for (auto const &value : props->dy) {
            dy_.push_back(StringUtils::FromString(DynamicUtils::DynamicToString(value)));
        }
        rotate_.clear();
        for (auto const &value : props->rotate) {
            rotate_.push_back(StringUtils::FromString(DynamicUtils::DynamicToString(value)));
        }
        if (props->inlineSize.empty()) {
            inlineSize_.reset();
        } else {
            inlineSize_ = StringUtils::FromString(DynamicUtils::DynamicToString(props->inlineSize));
        }

        if (props->textLength.empty()) {
            textLength_.reset();
        } else {
            textLength_ = StringUtils::FromString(DynamicUtils::DynamicToString(props->textLength));
        }

        hasBaselineShift_ = false;
        hasAlign_ = false;
        if (!props->verticalAlign.empty()) {
            auto align = DynamicUtils::DynamicToString(props->verticalAlign);
            align.erase(0, align.find_first_not_of(" \t\r\n"));
            align.erase(align.find_last_not_of(" \t\r\n") + 1);
            size_t i = align.find_last_of(' ');
            align_ = alignmentBaselineFromStr(align.substr(i));
            baselineShift_ = align.substr(0, i);
            hasAlign_ = true;
            hasBaselineShift_ = true;
        } else {
            align_ = AlignmentBaseline::baseline;
            baselineShift_.clear();
            hasAlign_ = false;
            hasBaselineShift_ = false;
        }

        if (!props->baselineShift.empty()) {
            baselineShift_ = DynamicUtils::DynamicToString(props->baselineShift);
            hasBaselineShift_ = true;
        }
        if (!props->alignmentBaseline.empty()) {
            align_ = alignmentBaselineFromStr(props->alignmentBaseline);
            hasAlign_ = true;
        }

        lengthAdjust_ = textLengthAdjustFromStr(props->lengthAdjust);
    }

    void SetGlyphContext(const std::shared_ptr<GlyphContext> &ctx) { glyphCtx_ = ctx; }
    
    void setAlignmentBaseline(const AlignmentBaseline &alignmentBaseline) { align_ = alignmentBaseline; }
    
    bool hasAlignmentBaseline() { return hasAlign_; }
    
    void setBaselineShift(const std::string &baselineShift) { baselineShift_ = baselineShift; }
    
    bool hasBaselineShift() { return hasBaselineShift_; }

protected:
    void InitGlyph(OH_Drawing_Canvas *canvas, double scale);

    std::vector<Dimension> x_;
    std::vector<Dimension> y_;
    std::vector<Dimension> dx_;
    std::vector<Dimension> dy_;
    std::vector<Dimension> rotate_;
    std::optional<Dimension> inlineSize_;
    std::optional<Dimension> textLength_;

    std::string baselineShift_;
    bool hasBaselineShift_ = false;
    TextLengthAdjust lengthAdjust_ = TextLengthAdjust::spacing;
    AlignmentBaseline align_ = AlignmentBaseline::baseline;
    bool hasAlign_ = false;

    std::shared_ptr<GlyphContext> glyphCtx_;
};

} // namespace svg
} // namespace rnoh