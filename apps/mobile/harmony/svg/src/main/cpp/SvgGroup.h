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

#pragma once

#include "SvgNode.h"
#include "properties/Font.h"
#include "FontHolderBase.h"
#include <native_drawing/drawing_canvas.h>

namespace rnoh {
namespace svg {

class SvgFilter; // 前置声明

class SvgGroup : public SvgNode, public FontHolderBase {

public:
    SvgGroup() : SvgNode() { InitGroupFlag(); }
    ~SvgGroup() override = default;

    Rect AsBounds() override;

protected:
    void OnDraw(OH_Drawing_Canvas *canvas) override;

private:
    // svg g, use
    void InitGroupFlag();
    void OnInitStyle() override;
    void ApplyFilterToChildren(std::shared_ptr<SvgFilter> filter);
    void RestoreChildrenFilter();
    // 用于保存子节点的原始filter设置
    std::vector<std::string> originalFilters_;
};

} // namespace svg
} // namespace rnoh
