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

#include "SvgSvg.h"
#include "properties/Size.h"
#include "utils/StringUtils.h"
#include "properties/ViewBox.h"
#include <native_drawing/drawing_rect.h>

namespace rnoh {
namespace svg {

namespace {
const int MOS_MEET = 0;
const int MOS_SLICE = 1;
const int MOS_NONE = 2;
} // namespace

SvgSvg::SvgSvg() : SvgGroup() {}

drawing::Path SvgSvg::AsPath() {
    drawing::Path path;
    for (const auto &child : children_) {
        auto childPath = child->AsPath();
        path.Union(childPath);
    }
    return path;
}

Size SvgSvg::GetSize() const { return {width_, height_}; }

Rect SvgSvg::GetViewBox() const {
    return Rect(svgAttribute_.vbX.ConvertToPx(scale_), svgAttribute_.vbY.ConvertToPx(scale_),
                svgAttribute_.vbWidth.ConvertToPx(scale_), svgAttribute_.vbHeight.ConvertToPx(scale_));
}

drawing::Matrix SvgSvg::FitCanvas(OH_Drawing_Canvas *canvas) {
    const auto svgSize = GetSize();
    const auto vbRect = GetViewBox(); // should be viewBox attribute
    const auto eRect = Rect(svgAttribute_.x.ConvertToPx(scale_), svgAttribute_.y.ConvertToPx(scale_), svgSize.Width(),
                            svgSize.Height());
    DLOG(INFO) << "[FitCanvas] viewBox = " << vbRect.ToString() << " svgSize = " << svgSize.ToString()
               << " canvas = " << OH_Drawing_CanvasGetWidth(canvas) << ", " << OH_Drawing_CanvasGetHeight(canvas);
    drawing::Rect clipRect(0.0f, 0.0f, svgSize.Width(), svgSize.Height());
    OH_Drawing_CanvasClipRect(canvas, clipRect.get(), OH_Drawing_CanvasClipOp::INTERSECT, true);
    drawing::Matrix transformMatrix;
    if (vbRect.IsValid()) {
        transformMatrix = ViewBox::getTransform(vbRect, eRect, svgAttribute_.align, svgAttribute_.meetOrSlice);
        OH_Drawing_CanvasConcatMatrix(canvas, transformMatrix.get());
    }
    return transformMatrix;
}

void SvgSvg::Draw(OH_Drawing_Canvas *canvas) {
    float width = OH_Drawing_CanvasGetWidth(canvas);
    float height = OH_Drawing_CanvasGetHeight(canvas);
    context_->SetCanvasBounds(Rect(0.0f, 0.0f, width, height));
    context_->SetRootViewBox(GetViewBox());
    width_ = width;
    height_ = height;
    DLOG(INFO) << "[svgView] bbHeight: " << svgAttribute_.height.Value();
    DLOG(INFO) << "[svgView] bbWidth: " << svgAttribute_.height.Value();
    DLOG(INFO) << "[svgView] svg Height: " << height_;
    DLOG(INFO) << "[svgView] svg Width: " << width_;
    DLOG(INFO) << "[svgView] canvas Height: " << height;
    DLOG(INFO) << "[svgView] canvas Width: " << width;
    context_->SetSvgSize(Size(width_, height_));
    // apply scale
    OH_Drawing_CanvasSave(canvas);
    auto canvasTransformMatrix = FitCanvas(canvas);
    constexpr size_t matrixScaleXIndex = 0;
    constexpr size_t matrixScaleYIndex = 4;
    context_->SetCanvasScale(canvasTransformMatrix.GetValue(matrixScaleXIndex),
                             canvasTransformMatrix.GetValue(matrixScaleYIndex));
    SvgNode::Draw(canvas);
    OH_Drawing_CanvasRestore(canvas);
};

std::shared_ptr<SvgNode> SvgSvg::HitTest(double x, double y) {
    const auto svgSize = GetSize();
    const auto vbRect = GetViewBox();
    const auto eRect = Rect(svgAttribute_.x.ConvertToPx(scale_), svgAttribute_.y.ConvertToPx(scale_), svgSize.Width(), svgSize.Height());
    
    drawing::Matrix transformMatrix;
    if (vbRect.IsValid()) {
        transformMatrix = ViewBox::getTransform(vbRect, eRect, svgAttribute_.align, svgAttribute_.meetOrSlice);
    }
    
    double localX = x;
    double localY = y;
    auto inverse = transformMatrix.Invert();
    if (inverse) {
        auto p = inverse->MapPoint(OH_Drawing_Point2D{static_cast<float>(x), static_cast<float>(y)});
        localX = p.x;
        localY = p.y;
    }
    return SvgNode::HitTest(localX, localY);
}

} // namespace svg
} // namespace rnoh
