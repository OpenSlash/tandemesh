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

#include "RNSVGFilterComponentInstance.h"
#include "RNSVGFeColorMatrixComponentInstance.h"
#include "RNSVGFeBlendComponentInstance.h"
#include "RNSVGFeCompositeComponentInstance.h"
#include "RNSVGFeFloodComponentInstance.h"
#include "RNSVGFeOffsetComponentInstance.h"
#include "RNSVGFeGaussianBlurComponentInstance.h"
#include "utils/FilterManager.h"
#include <sstream>
#include <utility>
#include <folly/dynamic.h>
#include <cmath>
#include <iomanip>
#include <string>
namespace rnoh {
namespace svg {

RNSVGFilterComponentInstance::RNSVGFilterComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgFilter);
}

void RNSVGFilterComponentInstance::onPropsChanged(SharedConcreteProps const &props) {}

void RNSVGFilterComponentInstance::UpdateElementProps() {
    m_svgFilter->UpdateCommonProps(m_props);
    // Set the name as the ID for the SvgNode using the UpdateHrefRenderProps method
    m_svgFilter->UpdateHrefRenderProps(m_props);
    parseFilterId();
    collectFilterEffectsInOrder();
}

void RNSVGFilterComponentInstance::parseFilterId() {
    if (m_props) {
        filterId_ = m_props->name;
    }
}

void RNSVGFilterComponentInstance::collectFilterEffectsInOrder()
{
    if (m_svgFilter) {
        m_svgFilter->ClearFilterEffects();
    }
    
    auto children = getChildren();
    if (collectFeDropShadowInfo()) {
        return;
    }

    for (const auto& child : children) {
        if (auto childInstance = std::dynamic_pointer_cast<ComponentInstance>(child)) {
            std::string childComponentName = childInstance->getComponentName();
            if (childComponentName.find("FeColorMatrix") != std::string::npos) {
                collectSingleFeColorMatrix(childInstance);
            } else if (childComponentName.find("FeFlood") != std::string::npos) {
                collectSingleFeFlood(childInstance);
            } else if (childComponentName.find("FeOffset") != std::string::npos) {
                collectSingleFeOffset(childInstance);
            } else if (childComponentName.find("FeComposite") != std::string::npos) {
                collectSingleFeComposite(childInstance);
            } else if (childComponentName.find("FeGaussianBlur") != std::string::npos) {
                collectSingleFeGaussianBlur(childInstance);
            } else if (childComponentName.find("FeBlend") != std::string::npos) {
                collectSingleFeBlend(childInstance);
            }
        }
    }
}

bool RNSVGFilterComponentInstance::collectFeDropShadowInfo()
{
    auto children = getChildren();

    std::shared_ptr<RNSVGFeGaussianBlurComponentInstance> blurInstance = nullptr;
    std::shared_ptr<RNSVGFeOffsetComponentInstance> offsetInstance = nullptr;
    std::shared_ptr<RNSVGFeFloodComponentInstance> floodInstance = nullptr;
    std::shared_ptr<RNSVGFeCompositeComponentInstance> compositeInstance = nullptr;

    for (const auto& child : children) {
        if (auto childInstance = std::dynamic_pointer_cast<ComponentInstance>(child)) {
            std::string childComponentName = childInstance->getComponentName();
            if (childComponentName.find("FeGaussianBlur") != std::string::npos) {
                blurInstance = std::dynamic_pointer_cast<RNSVGFeGaussianBlurComponentInstance>(childInstance);
            } else if (childComponentName.find("FeOffset") != std::string::npos) {
                offsetInstance = std::dynamic_pointer_cast<RNSVGFeOffsetComponentInstance>(childInstance);
            } else if (childComponentName.find("FeFlood") != std::string::npos) {
                floodInstance = std::dynamic_pointer_cast<RNSVGFeFloodComponentInstance>(childInstance);
            }
        }
    }
    
    if (blurInstance && offsetInstance && floodInstance) {
        std::string in = blurInstance->getIn();
        float dx = offsetInstance->getDx();
        float dy = offsetInstance->getDy();
        float stdDeviation = blurInstance->getStdDeviationX();
        std::string floodColorStr = floodInstance->getFloodColor();
        if (floodColorStr.empty()) {
            floodColorStr = "black";
        }
        float floodOpacity = floodInstance->getFloodOpacity();
        FilterEffect dropShadowEffect(FilterType::DROP_SHADOW, in, dx, dy, stdDeviation, floodColorStr, floodOpacity);
        m_svgFilter->AddFilterEffect(dropShadowEffect);
        return true;
    }
    return false;
}

void RNSVGFilterComponentInstance::collectSingleFeColorMatrix(std::shared_ptr<ComponentInstance> childInstance)
{
    if (auto feColorMatrixInstance = std::dynamic_pointer_cast<RNSVGFeColorMatrixComponentInstance>(childInstance)) {
        std::string in = feColorMatrixInstance->getIn();
        std::string type = feColorMatrixInstance->getType();
        std::vector<float> values = feColorMatrixInstance->getValues();
        std::string result = feColorMatrixInstance->getResult();
        FilterEffect colorMatrixEffect(FilterType::COLOR_MATRIX, values, in, type, result);
        m_svgFilter->AddFilterEffect(colorMatrixEffect);
    }
}

void RNSVGFilterComponentInstance::collectSingleFeFlood(std::shared_ptr<ComponentInstance> childInstance)
{
    auto feFloodInstance = std::dynamic_pointer_cast<RNSVGFeFloodComponentInstance>(childInstance);
    if (feFloodInstance) {
        std::string in = feFloodInstance->getIn();
        std::string floodColorStr = feFloodInstance->getFloodColor();
        if (floodColorStr.empty()) {
            floodColorStr = "black";
        }
        float floodOpacity = feFloodInstance->getFloodOpacity();
        std::string result = feFloodInstance->getResult();
        FilterEffect floodEffect(FilterType::FEFLOOD, in, floodColorStr, floodOpacity, result);
        m_svgFilter->AddFilterEffect(floodEffect);
    }
}

void RNSVGFilterComponentInstance::collectSingleFeOffset(std::shared_ptr<ComponentInstance> childInstance)
{
    if (auto feOffsetInstance = std::dynamic_pointer_cast<RNSVGFeOffsetComponentInstance>(childInstance)) {
        std::string in = feOffsetInstance->getIn();
        std::string result = feOffsetInstance->getResult();
        float dx = feOffsetInstance->getDx();
        float dy = feOffsetInstance->getDy();
        FilterEffect offsetEffect(FilterType::FEOFFSET, in, dx, dy, result);
        m_svgFilter->AddFilterEffect(offsetEffect);
    }
}

void RNSVGFilterComponentInstance::collectSingleFeComposite(std::shared_ptr<ComponentInstance> childInstance)
{
    if (auto feCompositeInstance = std::dynamic_pointer_cast<RNSVGFeCompositeComponentInstance>(childInstance)) {
        std::string in = feCompositeInstance->getIn();
        std::string in2 = feCompositeInstance->getIn2();
        std::string operatorType = feCompositeInstance->getOperator();
        std::string k1 = "0", k2 = "0", k3 = "0", k4 = "0";
        if (operatorType == "arithmetic") {
            k1 = feCompositeInstance->getK1();
            k2 = feCompositeInstance->getK2();
            k3 = feCompositeInstance->getK3();
            k4 = feCompositeInstance->getK4();
        }
        std::string result = feCompositeInstance->getResult();
        FilterEffect compositeEffect(FilterType::COMPOSITE, in, in2, operatorType, k1, k2, k3, k4, result);
        m_svgFilter->AddFilterEffect(compositeEffect);
    }
}

void RNSVGFilterComponentInstance::collectSingleFeGaussianBlur(std::shared_ptr<ComponentInstance> childInstance)
{
    if (auto feGaussianBlurInstance = std::dynamic_pointer_cast<RNSVGFeGaussianBlurComponentInstance>(childInstance)) {
        std::string in = feGaussianBlurInstance->getIn();
        float stdDeviationX = feGaussianBlurInstance->getStdDeviationX();
        float stdDeviationY = feGaussianBlurInstance->getStdDeviationY();
        std::string edgeMode = feGaussianBlurInstance->getEdgeMode();
        std::string result = feGaussianBlurInstance->getResult();
        FilterEffect gaussianBlurEffect(FilterType::GAUSSIAN_BLUR, in, stdDeviationX, stdDeviationY, edgeMode, result);
        m_svgFilter->AddFilterEffect(gaussianBlurEffect);
    }
}

void RNSVGFilterComponentInstance::collectSingleFeBlend(std::shared_ptr<ComponentInstance> childInstance)
{
    if (auto feBlendInstance = std::dynamic_pointer_cast<RNSVGFeBlendComponentInstance>(childInstance)) {
        std::string in = feBlendInstance->getIn();
        std::string in2 = feBlendInstance->getIn2();
        std::string mode = feBlendInstance->getMode();
        std::string result = feBlendInstance->getResult();
        FilterEffect blendEffect(FilterType::BLEND, in, in2, mode, result);
        m_svgFilter->AddFilterEffect(blendEffect);
    }
}
}
}