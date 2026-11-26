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

#include "SvgGroup.h"
#include "SvgFilter.h"
#include "drawing/Brush.h"

namespace rnoh {
namespace svg {

void SvgGroup::InitGroupFlag() {
    hrefFill_ = true;
    hrefRender_ = true;
    passStyle_ = true;
    inheritStyle_ = true;
    drawTraversed_ = true;
}

void SvgGroup::OnInitStyle() {
    if (!font_) {
        InitFont(GetScale());
    }
    for (auto &child : children_) {
		child->SetWholeOpacity(attributes_.selfOpacity);
        child->SetWholeFillOpacity(attributes_.fillState.GetOpacity());
        if (auto childG = std::dynamic_pointer_cast<FontHolderBase>(child)) {
            childG->InheritFont(font_, child->GetScale());
        }
    }
}

void SvgGroup::OnDraw(OH_Drawing_Canvas *canvas) {
    SvgNode::OnDraw(canvas);
}

void SvgGroup::ApplyFilterToChildren(std::shared_ptr<SvgFilter> filter) {
    if (!filter) {
        return;
    }
    originalFilters_.clear();
    
    for (auto& child : children_) {
        if (child) {
            originalFilters_.push_back(child->GetBaseAttributes().filterId);
            auto childAttrs = child->GetBaseAttributes();
            childAttrs.filterId = attributes_.filterId;
            childAttrs.hasInheritedFilter = true;
            child->SetBaseAttributes(childAttrs);
        }
    }
}

void SvgGroup::RestoreChildrenFilter() {
    for (size_t i = 0; i < children_.size() && i < originalFilters_.size(); ++i) {
        if (children_[i]) {
            auto childAttrs = children_[i]->GetBaseAttributes();
            childAttrs.filterId = originalFilters_[i];
            childAttrs.hasInheritedFilter = false;
            children_[i]->SetBaseAttributes(childAttrs);
        }
    }
    
    originalFilters_.clear();
}

Rect SvgGroup::AsBounds() {
    if (children_.empty()) {
        return Rect(0, 0, 0, 0);
    }
    
    auto bounds = children_[0]->AsBounds();
    for (size_t i = 1; i < children_.size(); ++i) {
        auto childBounds = children_[i]->AsBounds();
        // 合并边界
        float left = std::min(bounds.Left(), childBounds.Left());
        float top = std::min(bounds.Top(), childBounds.Top());
        float right = std::max(bounds.Right(), childBounds.Right());
        float bottom = std::max(bounds.Bottom(), childBounds.Bottom());
        bounds = Rect(left, top, right - left, bottom - top);
    }
    
    return bounds;
}

} // namespace svg
} // namespace rnoh