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

#include "RNSVGTextPathComponentInstance.h"
#include "utils/DynamicUtils.h"
#include "SvgTSpan.h"
#include <react/renderer/components/react_native_svg/Props.h>

namespace rnoh {
namespace svg {

RNSVGTextPathComponentInstance::RNSVGTextPathComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgTP);
}

void RNSVGTextPathComponentInstance::UpdateElementProps() {
    m_svgTP->UpdateCommonProps(m_props);
    m_svgTP->UpdateFontProps(m_props);
    m_svgTP->UpdateTextProps(m_props);
    m_svgTP->href_ = m_props->href;
    m_svgTP->startOffset_ = StringUtils::FromString(DynamicUtils::DynamicToString(m_props->startOffset)).ConvertToPx(m_layoutMetrics.pointScaleFactor);
    m_svgTP->spacing_ = textPathSpacingFromStr(m_props->spacing);
    m_svgTP->midLine_ = textPathMidLineFromStr(m_props->midLine);
    m_svgTP->side_ = textPathSideFromStr(m_props->side);
    m_svgTP->method_ = textPathMethodFromStr(m_props->method);
}
} // namespace svg
} // namespace rnoh
