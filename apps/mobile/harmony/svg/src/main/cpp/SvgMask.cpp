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

#include "SvgMask.h"
#include "drawing/Brush.h"
#include <native_drawing/drawing_color_filter.h>
#include <native_drawing/drawing_filter.h>
#include <native_drawing/drawing_canvas.h>
#include <native_drawing/drawing_color.h>

namespace rnoh {
namespace svg {

void SvgMask::OnDrawTraversedBefore(OH_Drawing_Canvas *canvas) {
    DLOG(INFO) << "[RNSVGMask] OnDrawTraversedBefore";
    // TODO implement proper support for units
    auto svgBounds = AsBounds();
    auto hostNodeBounds = GetHrefBounds();
    auto nodeBounds = (maskAttribute_.maskUnits == Unit::userSpaceOnUse)
                          ? svgBounds
                          : hostNodeBounds;
    DLOG(INFO) << "[RNSVGMask] Left0: " << nodeBounds.Left();
    DLOG(INFO) << "[RNSVGMask] Top0: " << nodeBounds.Top();
    DLOG(INFO) << "[RNSVGMask] Width0: " << nodeBounds.Width();
    DLOG(INFO) << "[RNSVGMask] Height0: " << nodeBounds.Height();
    auto left = static_cast<float>(ParseUnitsAttr(maskAttribute_.x, nodeBounds.Width()));
    auto top = static_cast<float>(ParseUnitsAttr(maskAttribute_.y, nodeBounds.Height()));
    auto width = static_cast<float>(ParseUnitsAttr(maskAttribute_.width, nodeBounds.Width()));
    auto height = static_cast<float>(ParseUnitsAttr(maskAttribute_.height, nodeBounds.Height()));
	if (maskAttribute_.maskUnits != Unit::userSpaceOnUse) {
        left = hostNodeBounds.Left() + left;
        top = hostNodeBounds.Top() + top;
    }
    drawing::Rect maskBounds(left, top, width + left, height + top);
    maskBounds_ = maskBounds;

    // create mask layer
    OH_Drawing_CanvasSaveLayer(canvas, maskBounds_.get(), nullptr);
    // ready to render mask content
    canvasLayerCount_ = OH_Drawing_CanvasGetSaveCount(canvas);

    // set mask
    auto outerFilter = OH_Drawing_ColorFilterCreateLuma();
    auto innerFilter = OH_Drawing_ColorFilterCreateSrgbGammaToLinear();
    auto colorFilter = OH_Drawing_ColorFilterCreateCompose(outerFilter, innerFilter);
    auto *filter = OH_Drawing_FilterCreate();
    OH_Drawing_FilterSetColorFilter(filter, colorFilter);

    drawing::Brush maskFilter;
    OH_Drawing_BrushSetFilter(maskFilter.get(), filter);
    OH_Drawing_CanvasSaveLayer(canvas, nullptr, maskFilter.get());
}

void SvgMask::OnDrawTraversedAfter(OH_Drawing_Canvas *canvas) {
    DLOG(INFO) << "[RNSVGMask] OnDrawTraversedAfter";
    OH_Drawing_CanvasRestoreToCount(canvas, canvasLayerCount_);
    // create content layer and render content
    drawing::Brush maskBrush;
    OH_Drawing_BrushSetBlendMode(maskBrush.get(), BLEND_MODE_SRC_IN);

    OH_Drawing_CanvasSaveLayer(canvas, maskBounds_.get(), maskBrush.get());
    OH_Drawing_CanvasClipRect(canvas, maskBounds_.get(), INTERSECT, true);
}


void SvgMask::OnInitStyle() { DLOG(INFO) << "[RNSVGMask] OnInitStyle"; }

double SvgMask::ParseUnitsAttr(const Dimension &attr, double value) {
    // percent and px
    if (attr.Unit() == DimensionUnit::PERCENT) {
        return value * attr.Value();
    }
    return attr.ConvertToPx(scale_);
}

} // namespace svg
} // namespace rnoh
