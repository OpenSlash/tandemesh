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

#include "RNSVGSymbolComponentInstance.h"
#include <react/renderer/components/react_native_svg/Props.h>

namespace rnoh {
namespace svg {

RNSVGSymbolComponentInstance::RNSVGSymbolComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {
    SetSvgNode(m_svgSymbol);
}

void RNSVGSymbolComponentInstance::UpdateElementProps() {
    // set attribute to svgSymbol.
    m_svgSymbol->UpdateCommonProps(m_props);
    m_svgSymbol->mMinX = m_props->minX;
    m_svgSymbol->mMinY = m_props->minY;
    m_svgSymbol->mVbWidth = m_props->vbWidth;
    m_svgSymbol->mVbHeight = m_props->vbHeight;
    m_svgSymbol->mMeetOrSlice = m_props->meetOrSlice;
    m_svgSymbol->mAlign = m_props->align;
}

} // namespace svg
} // namespace rnoh
