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

#include <iostream>
#include "SvgNode.h"

namespace rnoh {
namespace svg {
std::shared_ptr<SvgNode> SvgContext::GetSvgNodeById(const std::string &id) const {
    auto item = idMapper_.find(id);
    if (item != idMapper_.end()) {
        auto node = item->second.lock();
        if (node) {
            return node;
        }
    }
    return nullptr;
}

void SvgContext::PushStyle(const std::string &styleName, const std::pair<std::string, std::string> &attrPair) {
    const auto &arrMapIter = styleMap_.find(styleName);
    if (arrMapIter == styleMap_.end()) {
        AttrMap attrMap;
        attrMap.emplace(attrPair);
        styleMap_.emplace(std::make_pair(styleName, attrMap));
    } else {
        if (arrMapIter->second.find(attrPair.first) != arrMapIter->second.end()) {
            arrMapIter->second.erase(attrPair.first);
        }
        arrMapIter->second.emplace(attrPair);
    }
}

const AttrMap &SvgContext::GetAttrMap(const std::string &key) const {
    auto styleClassIter = styleMap_.find(key);
    if (styleClassIter != styleMap_.end()) {
        return styleClassIter->second;
    }
    static AttrMap emptyMap;
    return emptyMap;
}

const Rect &SvgContext::GetRootViewBox() const {
    if (!rootViewBox_.IsValid()) {
        static Rect rect;
        rect = Rect(0, 0, svgSize_.Width(), svgSize_.Height());
        return rect;
    }
    return rootViewBox_;
}

} // namespace svg
} // namespace rnoh
