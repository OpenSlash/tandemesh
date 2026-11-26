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

#include "RNSVGRadialGradientComponentInstance.h"
#include "utils/DynamicUtils.h"

namespace rnoh {
namespace svg {

RNSVGRadialGradientComponentInstance::RNSVGRadialGradientComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgRadialGradient);
}

void RNSVGRadialGradientComponentInstance::UpdateElementProps() {
    m_svgRadialGradient->UpdateHrefRenderProps(m_props);
    m_svgRadialGradient->SetAttrGradientUnits(m_props->gradientUnits); // first set attr gradientUnits
    m_svgRadialGradient->SetAttrFx(DynamicUtils::DynamicToString(m_props->fx));
    m_svgRadialGradient->SetAttrFy(DynamicUtils::DynamicToString(m_props->fy));
    m_svgRadialGradient->SetAttrCx(DynamicUtils::DynamicToString(m_props->cx));
    m_svgRadialGradient->SetAttrCy(DynamicUtils::DynamicToString(m_props->cy));
    m_svgRadialGradient->SetAttrRx(DynamicUtils::DynamicToString(m_props->rx));
    m_svgRadialGradient->SetAttrRy(DynamicUtils::DynamicToString(m_props->ry));
    m_svgRadialGradient->SetAttrGradient(m_props->gradient);
    m_svgRadialGradient->SetAttrGradientTransforms(m_props->gradientTransform);
}

} // namespace svg
} // namespace rnoh
