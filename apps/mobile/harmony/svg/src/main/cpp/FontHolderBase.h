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

#include <memory>
#include "properties/Font.h"

namespace rnoh {
namespace svg {

class FontHolderBase {
public:
    template <typename T> void UpdateFontProps(const std::shared_ptr<T> &props) {
        if constexpr (requires { props->font; }) {
            auto& font = props->font;
            if (!font.isNull()) {
                fp_.fontStyle = font.count("fontStyle") > 0 ? font["fontStyle"].asString() : "";
                fp_.fontVariant = font.count("fontVariant") > 0 ? font["fontVariant"].asString() : "";
                fp_.fontWeight = font.count("fontWeight") > 0 ? font["fontWeight"].asString() : "";
                fp_.fontStretch = font.count("fontStretch") > 0 ? font["fontStretch"].asString() : "";
                fp_.fontSize = font.count("fontSize") > 0 ? font["fontSize"].asString() : "";
                fp_.fontFamily = font.count("fontFamily") > 0 ? font["fontFamily"].asString() : "";
                fp_.textAnchor = font.count("textAnchor") > 0 ? font["textAnchor"].asString() : "";
                fp_.textDecoration = font.count("textDecoration") > 0 ? font["textDecoration"].asString() : "";
                fp_.letterSpacing = font.count("letterSpacing") > 0 ? font["letterSpacing"].asString() : "";
                fp_.wordSpacing = font.count("wordSpacing") > 0 ? font["wordSpacing"].asString() : "";
                fp_.kerning = font.count("kerning") > 0 ? font["kerning"].asString() : "";
                fp_.fontFeatureSettings = font.count("fontFeatureSettings") > 0 ? font["fontFeatureSettings"].asString() : "";
                fp_.fontVariantLigatures = font.count("fontVariantLigatures") > 0 ? font["fontVariantLigatures"].asString() : "";
                fp_.fontVariationSettings = font.count("fontVariationSettings") > 0 ? font["fontVariationSettings"].asString() : "";
            } else {
                fp_ = FontProps{};
            }
        }
    }

    void InheritFont(const std::shared_ptr<FontData> &parent, double scale);

    // init when font parent is not present
    void InitFont(double scale) { font_ = std::make_shared<FontData>(fp_, FontData{}, scale); }

protected:
    std::shared_ptr<FontData> font_;

private:
    FontProps fp_; // save intermediate font properties here because tree is not ready during onPropsChanged
};

} // namespace svg
} // namespace rnoh