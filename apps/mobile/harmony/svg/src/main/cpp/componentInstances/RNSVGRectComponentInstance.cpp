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

#include "RNSVGRectComponentInstance.h"
#include "utils/DynamicUtils.h"
#include <react/renderer/components/react_native_svg/Props.h>

namespace rnoh {
namespace svg {

RNSVGRectComponentInstance::RNSVGRectComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgRect);
}

void RNSVGRectComponentInstance::UpdateElementProps() {
    m_svgRect->UpdateCommonProps(m_props);
    m_svgRect->SetX(DynamicUtils::DynamicToString(m_props->x));
    m_svgRect->SetY(DynamicUtils::DynamicToString(m_props->y));
    m_svgRect->SetWidth(DynamicUtils::DynamicToString(m_props->width));
    m_svgRect->SetHeight(DynamicUtils::DynamicToString(m_props->height));
    std::string rx = DynamicUtils::DynamicToString(m_props->rx);
    if (rx.size() != 0) {
        m_svgRect->SetRX(rx);
    }
    std::string ry = DynamicUtils::DynamicToString(m_props->ry);
    if (ry.size() != 0) {
        m_svgRect->SetRY(ry);
    }
}

} // namespace svg
} // namespace rnoh
