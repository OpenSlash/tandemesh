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

#include "SvgFilter.h"
#include <native_drawing/drawing_mask_filter.h>
#include <native_drawing/drawing_color_filter.h>
#include <native_drawing/drawing_image_filter.h>
#include <native_drawing/drawing_shadow_layer.h>
#include <native_drawing/drawing_brush.h>
#include <native_drawing/drawing_pen.h>
#include <native_drawing/drawing_types.h>
#include <glog/logging.h>
#include <algorithm>
#include <cmath>
#include <string>
#include <stdexcept>
#include <sstream>
#include <vector>
#include <set>
#include <map>
#include <memory>

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

namespace rnoh {
namespace svg {

struct InputSource {
    std::string name;
    OH_Drawing_ImageFilter* imageFilter = nullptr;
    OH_Drawing_ColorFilter* colorFilter = nullptr;
    OH_Drawing_MaskFilter* maskFilter = nullptr;
    OH_Drawing_Filter* combinedFilter = nullptr;
    bool ownsFilters = false;
    
    ~InputSource() {
    }
    
    void cleanup() {
        if (ownsFilters) {
            if (imageFilter) {
                OH_Drawing_ImageFilterDestroy(imageFilter);
                imageFilter = nullptr;
            }
            if (colorFilter) {
                OH_Drawing_ColorFilterDestroy(colorFilter);
                colorFilter = nullptr;
            }
            if (maskFilter) {
                OH_Drawing_MaskFilterDestroy(maskFilter);
                maskFilter = nullptr;
            }
            if (combinedFilter) {
                OH_Drawing_FilterDestroy(combinedFilter);
                combinedFilter = nullptr;
            }
            ownsFilters = false;
        }
    }
};

class InputSourceManager {
private:
    std::map<std::string, std::shared_ptr<InputSource>> sources_;
    
public:
    void RegisterSource(const std::string& name, OH_Drawing_ImageFilter* imageFilter = nullptr, 
                       OH_Drawing_ColorFilter* colorFilter = nullptr, OH_Drawing_MaskFilter* maskFilter = nullptr) {
        auto source = std::make_shared<InputSource>();
        source->name = name;
        source->imageFilter = imageFilter;
        source->colorFilter = colorFilter;
        source->maskFilter = maskFilter;

        if (imageFilter || colorFilter || maskFilter) {
            source->combinedFilter = OH_Drawing_FilterCreate();
            if (source->combinedFilter) {
                if (imageFilter) OH_Drawing_FilterSetImageFilter(source->combinedFilter, imageFilter);
                if (colorFilter) OH_Drawing_FilterSetColorFilter(source->combinedFilter, colorFilter);
                if (maskFilter) OH_Drawing_FilterSetMaskFilter(source->combinedFilter, maskFilter);
            }
        }
        
        sources_[name] = source;
    }
    
    std::shared_ptr<InputSource> GetSource(const std::string& name) {
        if (name.empty()) {
            return nullptr;
        }
        
        auto it = sources_.find(name);
        if (it != sources_.end()) {
            return it->second;
        }

        if (name == "SourceGraphic") {
            return GetOrCreateStandardSource("SourceGraphic");
        } else if (name == "SourceAlpha") {
            return GetOrCreateStandardSource("SourceAlpha");
        } else if (name == "BackgroundImage") {
            return GetOrCreateStandardSource("BackgroundImage");
        } else if (name == "BackgroundAlpha") {
            return GetOrCreateStandardSource("BackgroundAlpha");
        } else if (name == "FillPaint") {
            return GetOrCreateStandardSource("FillPaint");
        } else if (name == "StrokePaint") {
            return GetOrCreateStandardSource("StrokePaint");
        }
        DLOG(WARNING) << "[SvgFilter] Input source not found: " << name;
        return nullptr;
    }
    
    void Clear() {
        for (auto& pair : sources_) {
            if (pair.second) {
                pair.second->cleanup();
            }
        }
        sources_.clear();
    }
    
private:
    std::shared_ptr<InputSource> GetOrCreateStandardSource(const std::string& name) {
        auto it = sources_.find(name);
        if (it != sources_.end()) {
            return it->second;
        }
        
        auto source = std::make_shared<InputSource>();
        source->name = name;
        if (name == "SourceAlpha") {
            float alphaMatrix[20] = {
                0, 0, 0, 0, 0,
                0, 0, 0, 0, 0,
                0, 0, 0, 0, 0,
                0, 0, 0, 1, 0
            };
            source->colorFilter = OH_Drawing_ColorFilterCreateMatrix(alphaMatrix);
        } else if (name == "BackgroundImage") {
        } else if (name == "BackgroundAlpha") {
            float backgroundAlphaMatrix[20] = {
                0, 0, 0, 0, 0,
                0, 0, 0, 0, 0,
                0, 0, 0, 0, 0,
                0, 0, 0, 1, 0
            };
            source->colorFilter = OH_Drawing_ColorFilterCreateMatrix(backgroundAlphaMatrix);
        } else if (name == "FillPaint" || name == "StrokePaint") {
            float matrix[20] = {
                1, 0, 0, 0, 0,
                0, 1, 0, 0, 0,
                0, 0, 1, 0, 0,
                0, 0, 0, 1, 0
            };
            source->colorFilter = OH_Drawing_ColorFilterCreateMatrix(matrix);
        }
        
        sources_[name] = source;
        return source;
    }
};

static InputSourceManager g_inputSourceManager;
OH_Drawing_ImageFilter* SvgFilter::GetInputImageFilter(const std::string& inputName) {
    if (inputName.empty() || inputName == "SourceGraphic") {
        return nullptr;
    }
    
    auto source = g_inputSourceManager.GetSource(inputName);
    return source ? source->imageFilter : nullptr;
}

OH_Drawing_ColorFilter* SvgFilter::GetInputColorFilter(const std::string& inputName) {
    if (inputName.empty() || inputName == "SourceGraphic") {
        return nullptr;
    }
    
    auto source = g_inputSourceManager.GetSource(inputName);
    if (source) {
        return source->colorFilter;
    } 
    return nullptr;
}

OH_Drawing_Filter* SvgFilter::GetInputFilter(const std::string& inputName) {
    if (inputName.empty() || inputName == "SourceGraphic") {
        return nullptr;
    }
    
    auto source = g_inputSourceManager.GetSource(inputName);
    return source ? source->combinedFilter : nullptr;
}

OH_Drawing_ColorFilter* SvgFilter::CombineWithInputColor(OH_Drawing_ColorFilter* newFilter, const std::string& inputName) {
    if (inputName.empty() || inputName == "SourceGraphic" || !newFilter) {
        return newFilter;
    }
    
    auto inputFilter = GetInputColorFilter(inputName);
    if (inputFilter) {
        return OH_Drawing_ColorFilterCreateCompose(newFilter, inputFilter);
    }
    
    return newFilter;
}

void SvgFilter::RegisterFilterResult(const std::string& resultId, OH_Drawing_ImageFilter* imageFilter, 
                                   OH_Drawing_ColorFilter* colorFilter, OH_Drawing_MaskFilter* maskFilter) {
    if (!resultId.empty()) {
        g_inputSourceManager.RegisterSource(resultId, imageFilter, colorFilter, maskFilter);
    }
}

void SvgFilter::AddFilterEffect(const FilterEffect& effect) {
    effects_.push_back(effect);
}

void SvgFilter::ApplyToBrush(rnoh::drawing::Brush& brush, const std::string& extra) {
    if (effects_.empty()) {
        return;
    }
    
    g_inputSourceManager.Clear();
    auto* mainFilter = OH_Drawing_FilterCreate();
    if (!mainFilter) {
        DLOG(WARNING) << "[SvgFilter] Failed to create main filter";
        return;
    }

    for (const auto& effect : effects_) {
        ApplySingleEffectToFilter(mainFilter, effect, &brush, nullptr, extra);
    }

    OH_Drawing_BrushSetFilter(brush.get(), mainFilter);
}

void SvgFilter::ApplyToPen(rnoh::drawing::Pen& pen) {
    if (effects_.empty()) {
        return;
    }

    g_inputSourceManager.Clear();
    auto* mainFilter = OH_Drawing_FilterCreate();
    if (!mainFilter) {
        DLOG(WARNING) << "[SvgFilter] Failed to create main filter";
        return;
    }
    
    for (const auto& effect : effects_) {
        ApplySingleEffectToFilter(mainFilter, effect, nullptr, &pen);
    }
    OH_Drawing_PenSetFilter(pen.get(), mainFilter);
}

void SvgFilter::ApplySingleEffectToFilter(OH_Drawing_Filter* filter, const FilterEffect& effect,
    rnoh::drawing::Brush* brush, rnoh::drawing::Pen* pen, const std::string& extra) {
    if (!filter) {
        DLOG(WARNING) << "[SvgFilter] Null filter passed to ApplySingleEffectToFilter";
        return;
    }
    switch (effect.filterType) {
        case FilterType::GAUSSIAN_BLUR: {
            if (extra == "image") {
                auto* imageFilter = CreateGaussianBlurFilter(effect);
                if (imageFilter) {
                    OH_Drawing_FilterSetImageFilter(filter, imageFilter);
                }
            } else {
                auto* maskFilter = CreateBlurFilter(effect);
                if (maskFilter) {
                    OH_Drawing_FilterSetMaskFilter(filter, maskFilter);
                }
            }
            break;
        }
        case FilterType::COLOR_MATRIX: {
            auto* colorFilter = CreateColorMatrixFilter(effect);
            if (colorFilter) {
                float identityMatrix[20] = {
                    1,0,0,0,0,  0,1,0,0,0,  0,0,1,0,0,  0,0,0,1,0
                };    
                OH_Drawing_ColorFilter* existingColorFilter = OH_Drawing_ColorFilterCreateMatrix(identityMatrix);
                OH_Drawing_FilterGetColorFilter(filter,existingColorFilter);
                if (existingColorFilter) {
                    auto* composedColorFilter = OH_Drawing_ColorFilterCreateCompose(colorFilter, existingColorFilter);
                    if (composedColorFilter) {
                        OH_Drawing_FilterSetColorFilter(filter, composedColorFilter);
                    }
                } else {
                    OH_Drawing_FilterSetColorFilter(filter, colorFilter);
                }
            }
            break;
        }
        case FilterType::FEFLOOD: {
            break;
        }
        case FilterType::FEOFFSET: {
            break;
        }
        case FilterType::BLUR: {
            auto* maskFilter = CreateBlurFilter(effect);
            if (maskFilter) {
                OH_Drawing_FilterSetMaskFilter(filter, maskFilter);
            }
            break;
        }
        case FilterType::DROP_SHADOW: {
            auto* shadowLayer = CreateDropShadowFilter(effect);
            if (shadowLayer) {
                if (brush) {
                    OH_Drawing_BrushSetShadowLayer(brush->get(), shadowLayer);
                } else if (pen) {
                    OH_Drawing_PenSetShadowLayer(pen->get(), shadowLayer);
                }
            }
            break;
        }
        case FilterType::BLEND: {
            break;
        }
        case FilterType::COMPOSITE: {
            break;
        }
        case FilterType::MERGE:
            break;
        default:
            DLOG(WARNING) << "[SvgFilter] Unsupported filter type: " << static_cast<int>(effect.filterType);
            break;
    }
}

OH_Drawing_ImageFilter* SvgFilter::CreateGaussianBlurFilter(const FilterEffect& effect) {
    auto inputFilter = GetInputImageFilter(effect.in);
    auto* blurFilter = OH_Drawing_ImageFilterCreateBlur(
        static_cast<double>(effect.stdDeviationX_),
        static_cast<double>(effect.stdDeviationY_),
        OH_Drawing_TileMode::CLAMP,
        inputFilter
    );

    if (!effect.result.empty()) {
        RegisterFilterResult(effect.result, blurFilter, nullptr, nullptr);
    }
    
    return blurFilter;
}

OH_Drawing_MaskFilter* SvgFilter::CreateBlurFilter(const FilterEffect& effect) {
    if (effect.values.size() >= 1) {
        auto* maskFilter = OH_Drawing_MaskFilterCreateBlur(
            OH_Drawing_BlurType::NORMAL,
            static_cast<double>(effect.values[0]),
            false
        );
        
        if (!effect.result.empty()) {
            RegisterFilterResult(effect.result, nullptr, nullptr, maskFilter);
        }
        
        return maskFilter;
    }
    return nullptr;
}

OH_Drawing_ColorFilter* SvgFilter::CreateColorMatrixFilter(const FilterEffect& effect) {
    OH_Drawing_ColorFilter* colorFilter = nullptr;
    if (effect.type == "matrix") {
        colorFilter = CreateMatrixFilter(effect.values);
    } else if (effect.type == "saturate") {
        colorFilter = CreateSaturateFilter(effect.values);
    } else if (effect.type == "hueRotate") {
        colorFilter = CreateHueRotateFilter(effect.values);
    } else if (effect.type == "luminanceToAlpha") {
        colorFilter = CreateLuminanceToAlphaFilter();
    } else {
        float identityMatrix[20] = {
            1,0,0,0,0,  0,1,0,0,0,  0,0,1,0,0,  0,0,0,1,0
        };
        colorFilter = OH_Drawing_ColorFilterCreateMatrix(identityMatrix);
    }
    
    colorFilter = CombineWithInputColor(colorFilter, effect.in);
    if (!effect.result.empty()) {
        RegisterFilterResult(effect.result, nullptr, colorFilter, nullptr);
    }
    return colorFilter;
}

OH_Drawing_ColorFilter* SvgFilter::CreateMatrixFilter(const std::vector<float>& values) {
    if (values.size() >= 20) {
        return OH_Drawing_ColorFilterCreateMatrix(values.data());
    } else {
        float identityMatrix[20] = {
            1,0,0,0,0,  0,1,0,0,0,  0,0,1,0,0,  0,0,0,1,0
        };
        return OH_Drawing_ColorFilterCreateMatrix(identityMatrix);
    }
}

OH_Drawing_ColorFilter* SvgFilter::CreateSaturateFilter(const std::vector<float>& values) {
    float saturation = values.empty() ? 1.0f : values[0];
    saturation = std::max(0.0f, saturation);
    const float rWeight = 0.213f;
    const float gWeight = 0.715f;
    const float bWeight = 0.072f;
    
    float colorMatrix[20] = {
        rWeight + (1.0f - rWeight) * saturation, gWeight * (1.0f - saturation), bWeight * (1.0f - saturation), 0.0f, 0.0f,
        rWeight * (1.0f - saturation), gWeight + (1.0f - gWeight) * saturation, bWeight * (1.0f - saturation), 0.0f, 0.0f,
        rWeight * (1.0f - saturation), gWeight * (1.0f - saturation), bWeight + (1.0f - bWeight) * saturation, 0.0f, 0.0f,
        0.0f, 0.0f, 0.0f, 1.0f, 0.0f
    };
    
    return OH_Drawing_ColorFilterCreateMatrix(colorMatrix);
}

OH_Drawing_ColorFilter* SvgFilter::CreateHueRotateFilter(const std::vector<float>& values) {
    float angle = values.empty() ? 0.0f : values[0];
    float radians = angle * M_PI / 180.0f;
    float cosA = cos(radians);
    float sinA = sin(radians);
    float colorMatrix[20] = {
        0.213f + 0.787f * cosA - 0.213f * sinA, 0.715f - 0.715f * cosA - 0.715f * sinA, 0.072f - 0.072f * cosA + 0.928f * sinA, 0.0f, 0.0f,
        0.213f - 0.213f * cosA + 0.143f * sinA, 0.715f + 0.285f * cosA + 0.140f * sinA, 0.072f - 0.072f * cosA - 0.283f * sinA, 0.0f, 0.0f,
        0.213f - 0.213f * cosA - 0.787f * sinA, 0.715f - 0.715f * cosA + 0.715f * sinA, 0.072f + 0.928f * cosA + 0.072f * sinA, 0.0f, 0.0f,
        0.0f, 0.0f, 0.0f, 1.0f, 0.0f
    };
    
    return OH_Drawing_ColorFilterCreateMatrix(colorMatrix);
}

OH_Drawing_ColorFilter* SvgFilter::CreateLuminanceToAlphaFilter() {
    float colorMatrix[20] = {
        0.0f, 0.0f, 0.0f, 0.0f, 0.0f,
        0.0f, 0.0f, 0.0f, 0.0f, 0.0f,
        0.0f, 0.0f, 0.0f, 0.0f, 0.0f,
        0.2125f, 0.7154f, 0.0721f, 0.0f, 0.0f
    };
    
    return OH_Drawing_ColorFilterCreateMatrix(colorMatrix);
}

OH_Drawing_ShadowLayer* SvgFilter::CreateDropShadowFilter(const FilterEffect& effect) {
    uint32_t shadowColor = 0xFF000000;
    if (!effect.floodColor.empty() && effect.floodColor != "none") {
        uint32_t baseColor = 0x000000;
        
        if (effect.floodColor.find("0x") == 0 || effect.floodColor.find("0X") == 0) {
            try {
                baseColor = static_cast<uint32_t>(std::stoul(effect.floodColor, nullptr, 16));
                if (effect.floodColor.length() > 8) {
                    baseColor = baseColor & 0x00FFFFFF;
                }
            } catch (const std::exception& e) {
                baseColor = 0x000000;
            }
        } else if (effect.floodColor.front() == '#') {
            std::string hex = effect.floodColor.substr(1);
            try {
                if (hex.length() == 6) {
                    baseColor = static_cast<uint32_t>(std::stoul(hex, nullptr, 16));
                } else if (hex.length() == 3) {
                    uint32_t r = std::stoul(hex.substr(0, 1), nullptr, 16);
                    uint32_t g = std::stoul(hex.substr(1, 1), nullptr, 16);
                    uint32_t b = std::stoul(hex.substr(2, 1), nullptr, 16);
                    baseColor = (r << 20) | (r << 16) | (g << 12) | (g << 8) | (b << 4) | b;
                }
            } catch (const std::exception& e) {
                baseColor = 0x000000;
            }
        }

        uint32_t red = (baseColor >> 16) & 0xFF;
        uint32_t green = (baseColor >> 8) & 0xFF;
        uint32_t blue = baseColor & 0xFF;
        float opacity = std::max(0.0f, std::min(1.0f, effect.floodOpacity));
        
        red = static_cast<uint32_t>(red * opacity + 255 * (1.0f - opacity));
        green = static_cast<uint32_t>(green * opacity + 255 * (1.0f - opacity));
        blue = static_cast<uint32_t>(blue * opacity + 255 * (1.0f - opacity));
        
        red = std::min(red, 255u);
        green = std::min(green, 255u);
        blue = std::min(blue, 255u);
        
        uint32_t alpha = 0xFF;
        shadowColor = (alpha << 24) | (red << 16) | (green << 8) | blue;
    }

    return OH_Drawing_ShadowLayerCreate(
        static_cast<double>(effect.stdDeviationX_),
        static_cast<double>(effect.dx_),
        static_cast<double>(effect.dy_),
        shadowColor
    );
}
}
}
