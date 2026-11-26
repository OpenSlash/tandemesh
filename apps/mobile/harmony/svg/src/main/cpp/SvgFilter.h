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

#include "SvgNode.h"
#include "drawing/Brush.h"
#include "drawing/Pen.h"
#include <native_drawing/drawing_canvas.h>
#include <native_drawing/drawing_filter.h>

namespace rnoh {
namespace svg {

enum class FilterType {
    BLUR,
    GAUSSIAN_BLUR,
    DROP_SHADOW,
    COLOR_MATRIX,
    FEOFFSET,
    FEFLOOD,
    BLEND,
    COMPOSITE,
    MERGE,
    UNKNOWN
};

struct FilterEffect {
    FilterType filterType = FilterType::UNKNOWN;
    std::vector<float> values;
    std::string type = "";
    std::string in = "SourceGraphic";
    std::string in2 = "";
    std::string result = "";

    std::string mode = "";
    std::string operator_ = "";
    std::string floodColor = "black";
    float floodOpacity = 1.0f;

    float dx_ = 0.0f;
    float dy_ = 0.0f;
    float stdDeviationX_ = 0.0f;
    float stdDeviationY_ = 0.0f;
    std::string edgeMode_ = "duplicate";
    std::string k1 = "";
    std::string k2 = "";
    std::string k3 = "";
    std::string k4 = "";

    FilterEffect() = default;
    FilterEffect(FilterType t, const std::vector<float>& v) : filterType(t), values(v) {}
    FilterEffect(FilterType t, const std::vector<float>& v, const std::string& inputSource, const std::string& inputType, const std::string& resultId) 
        : filterType(t), values(v), in(inputSource), type(inputType), result(resultId) {}
    FilterEffect(FilterType t, const std::string& inputSource, const std::string& inputSource2, const std::string& blendMode, const std::string& resultId)
        : filterType(t), in(inputSource), in2(inputSource2), mode(blendMode), result(resultId) {}
    FilterEffect(FilterType t, const std::string& inputSource, const std::string& color, float opacity, const std::string& resultId)
        : filterType(t), in(inputSource), floodColor(color), floodOpacity(opacity), result(resultId) {}
    FilterEffect(FilterType t, const std::string& inputSource, float dx, float dy, const std::string& resultId)
        : filterType(t), in(inputSource), dx_(dx), dy_(dy), result(resultId) {}
    FilterEffect(FilterType t, const std::string& inputSource, const std::string& inputSource2, const std::string& op,
        const std::string& k1_, const std::string& k2_, const std::string& k3_, const std::string& k4_, const std::string& resultId)
        : filterType(t), in(inputSource), in2(inputSource2), operator_(op), k1(k1_), k2(k2_), k3(k3_), k4(k4_), result(resultId) {}
    FilterEffect(FilterType t, const std::string& inputSource, float stdDeviationX, float stdDeviationY, const std::string& edgeMode, const std::string& resultId)
        : filterType(t), in(inputSource), stdDeviationX_(stdDeviationX), stdDeviationY_(stdDeviationY), edgeMode_(edgeMode), result(resultId) {}
    
    // DropShadow specific constructor
    FilterEffect(FilterType t, const std::string& inputSource, float dx, float dy, float stdDeviation, const std::string& color, float opacity)
        : filterType(t), in(inputSource), dx_(dx), dy_(dy), stdDeviationX_(stdDeviation), floodColor(color), floodOpacity(opacity) {}
};

class SvgFilter : virtual public SvgNode {
public:
    SvgFilter() : SvgNode() {}
    ~SvgFilter() noexcept override = default;
    void ApplyToBrush(rnoh::drawing::Brush& brush, const std::string& extra = "");
    void ApplyToPen(rnoh::drawing::Pen& pen);

    void AddFilterEffect(const FilterEffect& effect);
    void ClearFilterEffects() {
        effects_.clear();
    }

    void Draw(OH_Drawing_Canvas *canvas) override {}
private:
    std::vector<FilterEffect> effects_;
    void ApplySingleEffectToFilter(OH_Drawing_Filter* filter, const FilterEffect& effect,
        rnoh::drawing::Brush* brush = nullptr, rnoh::drawing::Pen* pen = nullptr, const std::string& extra = "");
    
    OH_Drawing_ImageFilter* CreateGaussianBlurFilter(const FilterEffect& effect);
    OH_Drawing_MaskFilter* CreateBlurFilter(const FilterEffect& effect);
    OH_Drawing_ColorFilter* CreateColorMatrixFilter(const FilterEffect& effect);
    OH_Drawing_ShadowLayer* CreateDropShadowFilter(const FilterEffect& effect);

    OH_Drawing_ColorFilter* CreateMatrixFilter(const std::vector<float>& values);
    OH_Drawing_ColorFilter* CreateSaturateFilter(const std::vector<float>& values);
    OH_Drawing_ColorFilter* CreateHueRotateFilter(const std::vector<float>& values);
    OH_Drawing_ColorFilter* CreateLuminanceToAlphaFilter();

    OH_Drawing_ImageFilter* GetInputImageFilter(const std::string& inputName);
    OH_Drawing_ColorFilter* GetInputColorFilter(const std::string& inputName);
    OH_Drawing_Filter* GetInputFilter(const std::string& inputName);
    OH_Drawing_ColorFilter* CombineWithInputColor(OH_Drawing_ColorFilter* newFilter, const std::string& inputName);
    void RegisterFilterResult(const std::string& resultId, OH_Drawing_ImageFilter* imageFilter = nullptr, 
                             OH_Drawing_ColorFilter* colorFilter = nullptr, OH_Drawing_MaskFilter* maskFilter = nullptr);
};

} // namespace svg
} // namespace rnoh
