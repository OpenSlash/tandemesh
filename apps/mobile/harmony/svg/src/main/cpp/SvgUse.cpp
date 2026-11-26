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

#include "SvgUse.h"

namespace rnoh {
namespace svg {

void SvgUse::OnDraw(OH_Drawing_Canvas *canvas) {
    auto svgContext = GetContext();
    if (!svgContext) {
        return;
    }

    auto refSvgNode = svgContext->GetSvgNodeById(href);
    if (!refSvgNode) {
        return;
    }

    if (x != 0 || y != 0) {
        OH_Drawing_CanvasTranslate(canvas, x * scale_, y * scale_);
    }

    AttributeScope scope(refSvgNode);
    std::shared_ptr<SvgSymbol> svgSymbol = std::dynamic_pointer_cast<SvgSymbol>(refSvgNode);

    refSvgNode->InheritUseAttr(attributes_);
    if (svgSymbol != nullptr) {
        svgSymbol->drawSymbol(canvas, width, height);
    } else {
        refSvgNode->Draw(canvas);
    }

    return;
}

drawing::Path SvgUse::AsPath() {
    drawing::Path path;
    auto svgContext = GetContext();
    if (!svgContext) {
        return path;
    }
    auto refSvgNode = svgContext->GetSvgNodeById(href);
    if (!refSvgNode) {
        return path;
    }
    path = refSvgNode->AsPath();
    return path;
}

SvgUse::AttributeScope::AttributeScope(const std::shared_ptr<SvgNode> &node) : node_(node) {
    auto nodeAttr = node->GetBaseAttributes();
    attributes_ = nodeAttr;
}

SvgUse::AttributeScope::~AttributeScope() {
    // TODO : check null ptr
    node_->SetBaseAttributes(attributes_);
}

} // namespace svg
} // namespace rnoh