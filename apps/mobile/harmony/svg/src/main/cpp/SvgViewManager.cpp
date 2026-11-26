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

#include "SvgViewManager.h"

namespace rnoh {
namespace svg {
void SvgViewManager::setSvgView(int tag, std::weak_ptr<SvgSvg> svg) {
    mTagToSvgViewMap.insert(std::make_pair(tag, svg));
    std::lock_guard lock(mTagtoSvgViewMapLock);
}

std::weak_ptr<SvgSvg> SvgViewManager::getSvgViewByTag(int tag) {
    std::lock_guard lock(mTagtoSvgViewMapLock);
    std::weak_ptr<SvgSvg> svgSvg;
    auto it = mTagToSvgViewMap.find(tag);
    if (it != mTagToSvgViewMap.end()) {
        svgSvg = it->second;
    }
    return svgSvg;
}

int32_t SvgViewManager::getHitNode(int32_t tag, double x, double y) {
    auto svg = getSvgViewByTag(tag).lock();
    if (!svg) {
        return -1;
    }
    
    auto hitNode = svg->HitTest(x, y);
    if (hitNode) {
        return hitNode->GetTag();
    }
    return -1;
}

void SvgViewManager::onDropView(int tag) {
    // 将svgview从容器中删除
    mTagToSvgViewMap.erase(tag);
    std::lock_guard lock(mTagtoSvgViewMapLock);
}

SvgViewManager::~SvgViewManager() {
    if (mTagToSvgViewMap.empty() == false) {
        mTagToSvgViewMap.clear();
    }
}

} // namespace svg
} // namespace rnoh