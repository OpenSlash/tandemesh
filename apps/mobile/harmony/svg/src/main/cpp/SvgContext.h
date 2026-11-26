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

#include <memory>
#include <string>
#include <unordered_map>
#include "properties/Rect.h"
#include "properties/Size.h"
#include "properties/Color.h"

namespace rnoh {
namespace svg {

using AttrMap = std::unordered_map<std::string, std::string>;
using ClassStyleMap = std::unordered_map<std::string, AttrMap>;
class SvgNode;
class SvgContext {
public:
    void Push(const std::string &value, const std::shared_ptr<SvgNode> &svgNode) { idMapper_.emplace(value, svgNode); }

    std::shared_ptr<SvgNode> GetSvgNodeById(const std::string &id) const;

    void PushStyle(const std::string &styleName, const std::pair<std::string, std::string> &attrPair);

    const AttrMap &GetAttrMap(const std::string &key) const;

    void SetRootViewBox(const Rect &viewBox) { rootViewBox_ = viewBox; }

    const Rect &GetRootViewBox() const;

    void SetSvgSize(const Size &size) { svgSize_ = size; }

    const Size &GetSvgSize() const { return svgSize_; }

    void SetSvgColor(const Color &svgColor) { svgColor_ = svgColor; }

    const Color &GetSvgColor() { return svgColor_; }

    void SetCanvasBounds(const Rect &canvasClipBounds) { canvasClipBounds_ = canvasClipBounds; }

    const Rect getCanvasBounds() { return canvasClipBounds_; }

    void SetCanvasScale(const double &canvasScaleX, const double &canvasScaleY) {
        canvasScaleX_ = canvasScaleX;
        canvasScaleY_ = canvasScaleY;
    }

    const double getCanvasScaleX() { return canvasScaleX_; }

    const double getCanvasScaleY() { return canvasScaleY_; }

    void ClearIdMap() { idMapper_.clear(); }

private:
    std::unordered_map<std::string, std::weak_ptr<SvgNode>> idMapper_;
    ClassStyleMap styleMap_;
    Rect rootViewBox_;
    Size svgSize_;
    Color svgColor_ = Color::WHITE; // 默认白色
    Rect canvasClipBounds_;
    double canvasScaleX_ = 1;
    double canvasScaleY_ = 1;
};

} // namespace svg
} // namespace rnoh