/*
 * Copyright (c) 2024 Huawei Device Co., Ltd. All rights reserved
 * Use of this source code is governed by a MIT license that can be
 * found in the LICENSE file.
 *
 * This file incorporates from the OpenHarmony project, licensed under
 * the Apache License, Version 2.0. Specifically:
 * - [OpenHarmony/arkui_ace_engine] (https://gitee.com/openharmony/arkui_ace_engine)
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License. You may obtain a copy
 * of the License at:
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 */

// from ArkUI "frameworks/core/components_ng/svg/parse/svg_node.h"
#pragma once

#include <memory>
#include <vector>
#include <unordered_set>
#include <type_traits>
#include "SvgBaseAttribute.h"
#include "SvgContext.h"
#include "SvgForeignProps.h"
#include "properties/Decoration.h"
#include "properties/Dimension.h"
#include "properties/Size.h"
#include <react/renderer/components/react_native_svg/Props.h>
#include "drawing/Path.h"
#include "utils/StringUtils.h"
#include "utils/SvgAttributesParser.h"
#include <glog/logging.h>
#include <multimedia/image_framework/image/pixelmap_native.h>
#include <native_drawing/drawing_canvas.h>
#include <native_drawing/drawing_rect.h>
#include "utils/DynamicUtils.h"

namespace rnoh {
namespace svg {

constexpr int INHERIT_TYPE = 2;

class SvgNode : public std::enable_shared_from_this<SvgNode> {
public:
    SvgNode() = default;
    virtual ~SvgNode() = default;

    std::shared_ptr<SvgContext> GetContext() { return context_; }
    void SetContext(std::shared_ptr<SvgContext> context) { context_ = context; }

    void ContextTraversal();
    void InitStyle(const SvgBaseAttribute &attr);

    virtual void Draw(OH_Drawing_Canvas *canvas);

    virtual drawing::Path AsPath() {
        DLOG(INFO) << "[SVGNode] AsPath";
        return drawing::Path();
    };

    virtual std::shared_ptr<SvgNode> HitTest(double x, double y);

    SvgBaseAttribute GetBaseAttributes() const { return attributes_; }

    void SetBaseAttributes(const SvgBaseAttribute &attr) { attributes_ = attr; }

    virtual void AppendChild(const std::shared_ptr<SvgNode> &child) { children_.emplace_back(child); }

    virtual void removeChild(const std::shared_ptr<SvgNode> &child) {
        auto it = std::find(children_.begin(), children_.end(), child);
        if (it != children_.end()) {
            children_.erase(it);
        }
    }

    template <typename ConcreteProps>
    void UpdateHrefRenderProps(const std::shared_ptr<ConcreteProps> &props, bool inherit = false) {
        if (!props) {
            return;
        }

        if (inherit) {
            attributes_.hasInheritedFilter = true;
        }
        // Check which properties are available and update them
        std::unordered_set<std::string> propSet;
        if constexpr (requires { props->propList; }) {
            propSet.insert(props->propList.begin(), props->propList.end());
        }

        // Always set name and display if available
        if constexpr (requires { props->name; }) {
            attributes_.id = props->name;
        }
        if constexpr (requires { props->display; }) {
            display_ = props->display != "none";
        }

        if (hrefRender_) {
            // Only access properties that exist in this Props type
            if constexpr (requires { props->matrix; }) {
                attributes_.transform = props->matrix;
            }
            if constexpr (requires { props->mask; }) {
                attributes_.maskId = props->mask;
            }
            if constexpr (requires { props->filter; } &&
                          std::is_same_v<std::decay_t<decltype(props->filter)>, std::string>) {
                attributes_.filterId = props->filter;
                attributes_.hasInheritedFilter = inherit;
            }
            if constexpr (requires { props->opacity; }) {
                attributes_.selfOpacity = props->opacity;
            }
            if constexpr (requires { props->markerStart; }) {
                attributes_.markerStart = props->markerStart;
            }
            if constexpr (requires { props->markerMid; }) {
                attributes_.markerMid = props->markerMid;
            }
            if constexpr (requires { props->markerEnd; }) {
                attributes_.markerEnd = props->markerEnd;
            }
            if constexpr (requires { props->clipPath; }) {
                if (!props->clipPath.empty()) {
                    attributes_.clipPath = props->clipPath;
                    hrefClipPath_ = props->clipPath;
                }
            }
        }
    }

    template <typename ConcreteProps>
    void UpdateCommonProps(const std::shared_ptr<ConcreteProps> &props, bool inherit = false) {
        UpdateHrefRenderProps(props, inherit);
        std::unordered_set<std::string> set;
        if constexpr (requires { props->propList; }) {
            for (const auto &prop : props->propList) {
                set.insert(prop);
            }
        }

        // Handle fill properties - support both object and string forms
        if constexpr (requires { props->fill; }) {
            bool hasFillProp = set.count("fill");
            if (props->fill.isString()) {
                auto fillValue = props->fill.asString();
                if (fillValue == "none") {
                    attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                } else if (fillValue == "currentColor") {
                    Color color = Color::TRANSPARENT;
                    color.SetUseCurrentColor(true);
                    attributes_.fillState.SetColor(color, true);
                } else {
                    attributes_.fillState.SetColor(SvgAttributesParser::GetColor(fillValue), hasFillProp);
                }
            } else if (props->fill.isInt() || props->fill.isDouble()) {
                uint32_t colorValue = props->fill.isInt() ? static_cast<uint32_t>(props->fill.asInt())
                                                          : static_cast<uint32_t>(props->fill.asDouble());
                if (facebook::react::isColorMeaningful(facebook::react::SharedColor(colorValue))) {
                    attributes_.fillState.SetColor(Color(colorValue), hasFillProp);
                } else {
                    attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                }
            } else if (!props->fill.isNull() && props->fill.isObject()) {
                if (props->fill.count("type") > 0) {
                    int fillType = props->fill["type"].asInt();
                    if (fillType == INHERIT_TYPE) {
                        Color color = Color::TRANSPARENT;
                        if (props->fill.count("payload") > 0 &&
                            (props->fill["payload"].isInt() || props->fill["payload"].isDouble())) {
                            uint32_t colorValue = props->fill["payload"].isInt()
                                                      ? static_cast<uint32_t>(props->fill["payload"].asInt())
                                                      : static_cast<uint32_t>(props->fill["payload"].asDouble());
                            color = Color(colorValue);
                        } else if (props->color) {
                            color = Color(*props->color);
                        }
                        color.SetUseCurrentColor(true);
                        attributes_.fillState.SetColor(color, true);
                    } else if (props->fill.count("payload") > 0) {
                        const auto &payload = props->fill["payload"];
                        if (payload.isString()) {
                            auto payloadValue = payload.asString();
                            if (payloadValue == "none") {
                                attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                            } else if (payloadValue == "currentColor") {
                                Color color = Color::TRANSPARENT;
                                color.SetUseCurrentColor(true);
                                attributes_.fillState.SetColor(color, true);
                            } else {
                                attributes_.fillState.SetColor(SvgAttributesParser::GetColor(payloadValue),
                                                               hasFillProp);
                            }
                        } else if (payload.isInt() || payload.isDouble()) {
                            uint32_t colorValue = payload.isInt() ? static_cast<uint32_t>(payload.asInt())
                                                                  : static_cast<uint32_t>(payload.asDouble());
                            if (facebook::react::isColorMeaningful(facebook::react::SharedColor(colorValue))) {
                                attributes_.fillState.SetColor(Color(colorValue), hasFillProp);
                            } else {
                                attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                            }
                        } else {
                            attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                        }
                    } else {
                        attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                    }
                } else if (props->fill.count("payload") > 0) {
                    const auto &payload = props->fill["payload"];
                    if (payload.isString()) {
                        auto payloadValue = payload.asString();
                        if (payloadValue == "none") {
                            attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                        } else if (payloadValue == "currentColor") {
                            Color color = Color::TRANSPARENT;
                            color.SetUseCurrentColor(true);
                            attributes_.fillState.SetColor(color, true);
                        } else {
                            attributes_.fillState.SetColor(SvgAttributesParser::GetColor(payloadValue), hasFillProp);
                        }
                    } else if (payload.isInt() || payload.isDouble()) {
                        uint32_t colorValue = payload.isInt() ? static_cast<uint32_t>(payload.asInt())
                                                              : static_cast<uint32_t>(payload.asDouble());
                        if (facebook::react::isColorMeaningful(facebook::react::SharedColor(colorValue))) {
                            attributes_.fillState.SetColor(Color(colorValue), hasFillProp);
                        } else {
                            attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                        }
                    } else {
                        attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                    }
                } else {
                    attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
                }
                if (props->fill.count("brushRef") > 0) {
                    attributes_.fillState.SetHref(props->fill["brushRef"].asString());
                }
            } else {
                attributes_.fillState.SetColor(Color::TRANSPARENT, hasFillProp);
            }
        }

        // Handle stroke properties - support both object and string forms
        if constexpr (requires { props->stroke; }) {
            bool hasStrokeProp = set.count("stroke");
            if (props->stroke.isString()) {
                auto strokeValue = props->stroke.asString();
                if (strokeValue == "none") {
                    attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
                } else if (strokeValue == "currentColor") {
                    Color color = Color::TRANSPARENT;
                    color.SetUseCurrentColor(true);
                    attributes_.strokeState.SetColor(color, true);
                } else {
                    attributes_.strokeState.SetColor(SvgAttributesParser::GetColor(strokeValue), hasStrokeProp);
                }
            } else if (props->stroke.isInt() || props->stroke.isDouble()) {
                uint32_t colorValue = props->stroke.isInt() ? static_cast<uint32_t>(props->stroke.asInt())
                                                            : static_cast<uint32_t>(props->stroke.asDouble());
                if (facebook::react::isColorMeaningful(facebook::react::SharedColor(colorValue))) {
                    attributes_.strokeState.SetColor(Color(colorValue), hasStrokeProp);
                } else {
                    attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
                }
            } else if (!props->stroke.isNull() && props->stroke.isObject()) {
                if (props->stroke.count("type") > 0) {
                    int strokeType = props->stroke["type"].asInt();
                    if (strokeType == INHERIT_TYPE) {
                        Color color = Color::TRANSPARENT;
                        if (props->stroke.count("payload") > 0 &&
                            (props->stroke["payload"].isInt() || props->stroke["payload"].isDouble())) {
                            uint32_t colorValue = props->stroke["payload"].isInt()
                                                      ? static_cast<uint32_t>(props->stroke["payload"].asInt())
                                                      : static_cast<uint32_t>(props->stroke["payload"].asDouble());
                            color = Color(colorValue);
                        } else if (props->color) {
                            color = Color(*props->color);
                        }
                        color.SetUseCurrentColor(true);
                        attributes_.strokeState.SetColor(color, true);
                    } else if (props->stroke.count("payload") > 0) {
                        const auto &payload = props->stroke["payload"];
                        if (payload.isString()) {
                            auto payloadValue = payload.asString();
                            if (payloadValue == "none") {
                                attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
                            } else if (payloadValue == "currentColor") {
                                Color color = Color::TRANSPARENT;
                                color.SetUseCurrentColor(true);
                                attributes_.strokeState.SetColor(color, true);
                            } else {
                                attributes_.strokeState.SetColor(SvgAttributesParser::GetColor(payloadValue),
                                                                 hasStrokeProp);
                            }
                        } else if (payload.isInt() || payload.isDouble()) {
                            uint32_t colorValue = payload.isInt() ? static_cast<uint32_t>(payload.asInt())
                                                                  : static_cast<uint32_t>(payload.asDouble());
                            if (facebook::react::isColorMeaningful(facebook::react::SharedColor(colorValue))) {
                                attributes_.strokeState.SetColor(Color(colorValue), hasStrokeProp);
                            } else {
                                attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
                            }
                        } else {
                            attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
                        }
                    }
                } else if (props->stroke.count("payload") > 0) {
                    const auto &payload = props->stroke["payload"];
                    if (payload.isString()) {
                        auto payloadValue = payload.asString();
                        if (payloadValue == "none") {
                            attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
                        } else if (payloadValue == "currentColor") {
                            Color color = Color::TRANSPARENT;
                            color.SetUseCurrentColor(true);
                            attributes_.strokeState.SetColor(color, true);
                        } else {
                            attributes_.strokeState.SetColor(SvgAttributesParser::GetColor(payloadValue),
                                                             hasStrokeProp);
                        }
                    } else if (payload.isInt() || payload.isDouble()) {
                        uint32_t colorValue = payload.isInt() ? static_cast<uint32_t>(payload.asInt())
                                                              : static_cast<uint32_t>(payload.asDouble());
                        if (facebook::react::isColorMeaningful(facebook::react::SharedColor(colorValue))) {
                            attributes_.strokeState.SetColor(Color(colorValue), hasStrokeProp);
                        } else {
                            attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
                        }
                    } else {
                        attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
                    }
                } else {
                    attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
                }
                if (props->stroke.count("brushRef") > 0) {
                    attributes_.strokeState.SetHref(props->stroke["brushRef"].asString());
                }
            } else {
                attributes_.strokeState.SetColor(Color::TRANSPARENT, hasStrokeProp);
            }
        }

        if constexpr (requires { props->fillOpacity; }) {
            attributes_.fillState.SetOpacity(std::clamp(props->fillOpacity, 0.0, 1.0), true);
        }
        if constexpr (requires { props->fillRule; }) {
            attributes_.fillState.SetFillRule(static_cast<FillState::FillRule>(props->fillRule), true);
        }
        if constexpr (requires { props->strokeWidth; }) {
            bool hasStrokeWidthInPropList = set.count("strokeWidth") > 0;
            if (hasStrokeWidthInPropList) {
                attributes_.strokeState.SetLineWidth(
                    StringUtils::StringToDouble(DynamicUtils::DynamicToString(props->strokeWidth)) * scale_, true);
            }
        }
        if constexpr (requires { props->strokeDasharray; }) {
            attributes_.strokeState.SetStrokeDashArray(
                StringUtils::stringVectorToDoubleVector(DynamicUtils::DynamicToStringVector(props->strokeDasharray),
                                                        scale_),
                set.count("strokeDasharray"));
        }
        if constexpr (requires { props->strokeDashoffset; }) {
            attributes_.strokeState.SetStrokeDashOffset(props->strokeDashoffset * scale_,
                                                        set.count("strokeDashoffset"));
        }
        if constexpr (requires { props->strokeLinecap; }) {
            attributes_.strokeState.SetLineCap(
                SvgAttributesParser::GetLineCapStyle(std::to_string(props->strokeLinecap)), set.count("strokeLinecap"));
        }
        if constexpr (requires { props->strokeLinejoin; }) {
            attributes_.strokeState.SetLineJoin(
                SvgAttributesParser::GetLineJoinStyle(std::to_string(props->strokeLinejoin)),
                set.count("strokeLinejoin"));
        }
        if constexpr (requires { props->vectorEffect; }) {
            attributes_.strokeState.SetVectorEffect(props->vectorEffect);
        }
        if constexpr (requires { props->strokeMiterlimit; }) {
            auto limit = props->strokeMiterlimit * scale_;
            if (GreatOrEqual(limit, 1.0)) {
                attributes_.strokeState.SetMiterLimit(limit, set.count("strokeMiterlimit"));
            }
        }
        if constexpr (requires { props->strokeOpacity; }) {
            attributes_.strokeState.SetOpacity(std::clamp(props->strokeOpacity, 0.0, 1.0), set.count("strokeOpacity"));
        }
        if constexpr (requires { props->clipRule; }) {
            attributes_.clipState.SetClipRule(static_cast<ClipState::ClipRule>(props->clipRule), true);
        }
    }

    virtual Rect AsBounds();

    void SetScale(const double &scale) { scale_ = scale; }

    double GetScale() const { return scale_; }
	
	void SetWholeOpacity(const double &wholeOpacity) { 
        DLOG(INFO) << "[svgView] SetWholeOpacity: " << wholeOpacity;
        wholeOpacity_ = wholeOpacity; }
    
    double GetWholeOpacity() const { return wholeOpacity_; }
    
    void SetWholeFillOpacity(const double &wholeFillOpacity) { 
        DLOG(INFO) << "[svgView] SetWholeFillOpacity: " << wholeFillOpacity;
        wholeFillOpacity_ = wholeFillOpacity; }
    
    double GetWholeFillOpacity() const { return wholeFillOpacity_; }

    void InheritAttr(const SvgBaseAttribute &parent) {
        attributes_.Inherit(parent);
        // svg color -> current color
        if (attributes_.strokeState.GetColor().IsUseCurrentColor()) {
            if (context_) {
                attributes_.strokeState.SetColor(context_->GetSvgColor(), true);
            }
        }
        if (attributes_.fillState.GetColor().IsUseCurrentColor()) {
            if (context_) {
                attributes_.fillState.SetColor(context_->GetSvgColor(), true);
            }
        }
    }

    void InheritUseAttr(const SvgBaseAttribute &parent) { attributes_.InheritFromUse(parent); }

    drawing::Matrix lastCanvasMatrix_;

    double relativeOnWidth(Dimension length);
    double relativeOnHeight(Dimension length);
    double relativeOnOther(Dimension length);

    double getCanvasWidth();
    double getCanvasHeight();
    double getCanvasDiagonal();

    void SetForeignObject(ForeignProps foreignProps) { _foreignPropsArray.emplace_back(foreignProps); }

    void SetTag(int32_t tag) { reactTag_ = tag; }
    int32_t GetTag() const { return reactTag_; }
    
    bool GetDisplay() const { return display_; }


protected:
    // override as need by derived class
    // called by function AppendChild
    virtual void OnAppendChild(const std::shared_ptr<SvgNode> &child) {}
    // called by function InitStyle
    virtual void OnInitStyle() {}

    virtual void OnDraw(OH_Drawing_Canvas *canvas) {}
    virtual void OnDrawTraversed(OH_Drawing_Canvas *canvas);
    void OnClipPath(OH_Drawing_Canvas *canvas);
    void OnMask(OH_Drawing_Canvas *canvas);
    void OnTransform(OH_Drawing_Canvas *canvas, std::vector<double> transform);

    const Rect &GetRootViewBox() const;

    void SetSmoothEdge(float edge) { attributes_.smoothEdge = edge; }
    float GetSmoothEdge() const { return attributes_.smoothEdge; }

    std::optional<Gradient> GetGradient(const std::string &href);

    std::shared_ptr<PatternAttr> GetPatternAttr(const std::string &href);
    void DrawForeignPixelMap(OH_Drawing_Canvas *canvas, ForeignProps _foreignProps);
    void DrawForeignMask(OH_Drawing_Canvas *canvas, const std::string &id);
    void DrawForeignClip(OH_Drawing_Canvas *canvas, const std::string &id, int clipRule);

    void InitNoneFlag() {
        hrefFill_ = false;
        hrefRender_ = false;
        passStyle_ = false;
        inheritStyle_ = false;
        drawTraversed_ = false;
    }

    SvgBaseAttribute attributes_;

    std::shared_ptr<SvgContext> context_;

    std::vector<std::shared_ptr<SvgNode>> children_;

    std::string hrefClipPath_;
    std::string imagePath_;

    double scale_ = 3.25;
	double wholeOpacity_ = 1.0;
	double wholeFillOpacity_ = 1.0;

    drawing::Matrix cTM_;

    bool display_ = true;

    bool hrefFill_ = true;      // get fill attributes from reference
    bool hrefRender_ = true;    // get render attr (mask, filter, transform, opacity,
                                // clip path) from reference
    bool passStyle_ = false;    // pass style attributes to child node, TAGS
                                // circle/path/line/... = false
    bool inheritStyle_ = true;  // inherit style attributes from parent node, TAGS
                                // mask/defs/pattern/filter = false
    bool drawTraversed_ = true; // enable OnDraw, TAGS mask/defs/pattern/filter = false

    double canvasHeight_ = -1;

    double canvasWidth_ = -1;

    double canvasDiagonal_ = -1;

    /*
      N[1/Sqrt[2], 36]
      The inverse of the square root of 2.
      Provide enough digits for the 128-bit IEEE quad (36 significant digits).
  */
    const double M_SQRT1_2l = 0.707106781186547524400844362104849039;

    std::vector<ForeignProps> _foreignPropsArray;
    
    int32_t reactTag_ = -1;
};

} // namespace svg
} // namespace rnoh
