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

#include "RNSVGLinearGradientComponentInstance.h"
#include "utils/DynamicUtils.h"

namespace rnoh {
namespace svg {

RNSVGLinearGradientComponentInstance::RNSVGLinearGradientComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgLinearGradient);
}

void RNSVGLinearGradientComponentInstance::UpdateElementProps() {
    m_svgLinearGradient->UpdateHrefRenderProps(m_props);
    m_svgLinearGradient->SetAttrGradientUnits(m_props->gradientUnits); // first set attr gradientUnits
    m_svgLinearGradient->SetAttrX1(DynamicUtils::DynamicToString(m_props->x1));
    m_svgLinearGradient->SetAttrY1(DynamicUtils::DynamicToString(m_props->y1));
    m_svgLinearGradient->SetAttrX2(DynamicUtils::DynamicToString(m_props->x2));
    m_svgLinearGradient->SetAttrY2(DynamicUtils::DynamicToString(m_props->y2));
    m_svgLinearGradient->SetAttrGradient(m_props->gradient);
    m_svgLinearGradient->SetAttrGradientTransforms(m_props->gradientTransform);
}

} // namespace svg
} // namespace rnoh
