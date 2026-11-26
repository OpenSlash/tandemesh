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

#include "RNSVGUseComponentInstance.h"
#include "utils/DynamicUtils.h"


namespace rnoh {
namespace svg {

RNSVGUseComponentInstance::RNSVGUseComponentInstance(Context context) : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgUse);
}

void RNSVGUseComponentInstance::UpdateElementProps() {
    m_svgUse->UpdateCommonProps(m_props);
    std::string xStr = DynamicUtils::DynamicToString(m_props->x);
    std::string yStr = DynamicUtils::DynamicToString(m_props->y);
    std::string widthStr = DynamicUtils::DynamicToString(m_props->width);
    std::string heightStr = DynamicUtils::DynamicToString(m_props->height);
    m_svgUse->x = xStr.empty() ? 0.0f : std::stof(xStr);
    m_svgUse->y = yStr.empty() ? 0.0f : std::stof(yStr);
    m_svgUse->width = widthStr.empty() ? 0.0f : std::stof(widthStr);
    m_svgUse->height = heightStr.empty() ? 0.0f : std::stof(heightStr);
    m_svgUse->href = m_props->href;
}

} // namespace svg
} // namespace rnoh
