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
#include "SvgGraphic.h"

namespace rnoh {
namespace svg {

class SvgDefs : public SvgNode {
public:
    SvgDefs() : SvgNode() { InitDefsFlag(); }
    ~SvgDefs() override = default;

    void InitDefsFlag() {
        hrefFill_ = false;
        hrefRender_ = false;
        inheritStyle_ = false;
        drawTraversed_ = false;
    }

    drawing::Path AsPath() override {
        drawing::Path path;
        // Early return if there are no children
        if (children_.empty()) {
            return path;
        }
        for (const auto& child : children_) {
            if (!child) {
                DLOG(WARNING) << "[SvgDefs:AsPath] Encountered a null child node";
                continue;
            }
            auto childPath = child->AsPath();
            path.Union(childPath);
        }
        return path;
    }
};

} // namespace svg
} // namespace rnoh
