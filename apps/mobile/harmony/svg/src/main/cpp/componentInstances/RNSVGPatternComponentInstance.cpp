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

#include "RNSVGPatternComponentInstance.h"
#include "utils/DynamicUtils.h"
#include "utils/SvgAttributesParser.h"
#include <react/renderer/components/react_native_svg/Props.h>

namespace rnoh {
namespace svg {

RNSVGPatternComponentInstance::RNSVGPatternComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgPattern);
}

void RNSVGPatternComponentInstance::UpdateElementProps() {
    // set attribute to svgPattern
    m_svgPattern->UpdateCommonProps(m_props);
    m_svgPattern->setPatternX(SvgAttributesParser::ParseDimension(DynamicUtils::DynamicToString(m_props->x), true));
    m_svgPattern->setPatternY(SvgAttributesParser::ParseDimension(DynamicUtils::DynamicToString(m_props->y), true));
    m_svgPattern->setWidth(SvgAttributesParser::ParseDimension(DynamicUtils::DynamicToString(m_props->width), true));
    m_svgPattern->setHeight(SvgAttributesParser::ParseDimension(DynamicUtils::DynamicToString(m_props->height), true));
    m_svgPattern->setPatternUnits(m_props->patternUnits);
    m_svgPattern->setPatternContentUnits(m_props->patternContentUnits);
    m_svgPattern->setViewBox(Rect(m_props->minX, m_props->minY, m_props->vbWidth, m_props->vbHeight));
    m_svgPattern->setmMinX(m_props->minX);
    m_svgPattern->setmMinY(m_props->minY);
    m_svgPattern->setmVbWidth(m_props->vbWidth);
    m_svgPattern->setmVbHeight(m_props->vbHeight);
    m_svgPattern->setmAlign(m_props->align);
    m_svgPattern->setmMeetOrSlice(m_props->meetOrSlice);
    m_svgPattern->setPatternTransforms(m_props->patternTransform);
}

} // namespace svg
} // namespace rnoh
