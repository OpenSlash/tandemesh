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

#include "RNSVGPathComponentInstance.h"
#include <react/renderer/components/react_native_svg/Props.h>

namespace rnoh {
namespace svg {

RNSVGPathComponentInstance::RNSVGPathComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgPath);
}

void RNSVGPathComponentInstance::UpdateElementProps() {
    DLOG(INFO) << "[RNSVGPathCI] d: " << m_props->d;
    DLOG(INFO) << "[RNSVGPathCI] pointScaleFactor: " << m_layoutMetrics.pointScaleFactor;
    m_svgPath->UpdateCommonProps(m_props);
    // Check if path data or scale factor has changed
    if (m_cacheD != m_props->d || m_cacheScale != m_layoutMetrics.pointScaleFactor) {
        m_svgPath->setD(m_props->d); // Update path data
        m_cacheD = m_props->d;       // Cache the new path data
        m_cacheScale = m_layoutMetrics.pointScaleFactor; // Cache the new scale factor
    } else {
        LOG(INFO) << "[RNSVGPathCI] No changes detected. Skipping update.";
    }
}

} // namespace svg
} // namespace rnoh
