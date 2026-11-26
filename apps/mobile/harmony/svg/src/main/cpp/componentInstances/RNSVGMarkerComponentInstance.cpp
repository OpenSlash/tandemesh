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

#include "RNSVGMarkerComponentInstance.h"
#include "utils/DynamicUtils.h"

namespace rnoh {
namespace svg {

RNSVGMarkerComponentInstance::RNSVGMarkerComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgMarker);
}

void RNSVGMarkerComponentInstance::UpdateElementProps() {
    m_svgMarker->UpdateCommonProps(m_props);
    m_svgMarker->setRefX(DynamicUtils::DynamicToString(m_props->refX));
    m_svgMarker->setRefY(DynamicUtils::DynamicToString(m_props->refY));
    m_svgMarker->setMarkerWidth(DynamicUtils::DynamicToString(m_props->markerWidth));
    m_svgMarker->setMarkerHeight(DynamicUtils::DynamicToString(m_props->markerHeight));
    m_svgMarker->setMarkerUnits(m_props->markerUnits);
    m_svgMarker->setOrient(m_props->orient);
    m_svgMarker->setMinX(m_props->minX);
    m_svgMarker->setMinY(m_props->minY);
    m_svgMarker->setVbWidth(m_props->vbWidth);
    m_svgMarker->setVbHeight(m_props->vbHeight);
    m_svgMarker->setAlign(m_props->align);
    m_svgMarker->setMeetOrSlice(m_props->meetOrSlice);
}

} // namespace svg
} // namespace rnoh
