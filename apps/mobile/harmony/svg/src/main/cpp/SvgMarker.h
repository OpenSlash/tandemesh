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

#include "SvgQuote.h"
#include "properties/Dimension.h"
#include "utils/SvgAttributesParser.h"
#include "utils/SvgMarkerPositionUtils.h"
#include <sstream>
#include <string>
#include <native_drawing/drawing_register_font.h>
#include <native_drawing/drawing_text_typography.h>
#include <native_drawing/drawing_matrix.h>

namespace rnoh {
namespace svg {

class SvgMarker : public SvgQuote {
public:
    SvgMarker() = default;
    ~SvgMarker() override = default;

    void setRefX(const std::string &refX) { markerAttribute_.refX = SvgAttributesParser::ParseDimension(refX, true); }

    void setRefY(const std::string &refY) { markerAttribute_.refY = SvgAttributesParser::ParseDimension(refY, true); }

    void setMarkerWidth(const std::string &markerWidth) {
        markerAttribute_.markerWidth = SvgAttributesParser::ParseDimension(markerWidth, true);
    }

    void setMarkerHeight(const std::string &markerHeight) {
        markerAttribute_.markerHeight = SvgAttributesParser::ParseDimension(markerHeight, true);
    }

    void setMarkerUnits(const std::string &markerUnits) { markerAttribute_.markerUnits = markerUnits; }

    void setOrient(const std::string &orient) { markerAttribute_.orient = orient; }

    void setMinX(double minX) { markerAttribute_.minX = Dimension(minX, DimensionUnit::VP); }

    void setMinY(double minY) { markerAttribute_.minY = Dimension(minY, DimensionUnit::VP); }

    void setVbWidth(double vbWidth) { markerAttribute_.vbWidth = Dimension(vbWidth, DimensionUnit::VP); }

    void setVbHeight(double vbHeight) { markerAttribute_.vbHeight = Dimension(vbHeight, DimensionUnit::VP); }

    void setAlign(std::string align) { markerAttribute_.align = align; }

    void setMeetOrSlice(int meetOrSlice) { markerAttribute_.meetOrSlice = meetOrSlice; }

    void renderMarker(OH_Drawing_Canvas *canvas, const SvgMarkerPosition &position, float strokeWidth);

    /**
     * from VirtualView.saveAndSetupCanvas
     *
     * Sets up the transform matrix on the canvas before an element is drawn.
     *
     * <p>NB: for perf reasons this does not apply opacity, as that would mean creating a new canvas
     * layer (which allocates an offscreen bitmap) and having it composited afterwards. Instead, the
     * drawing code should apply opacity recursively.
     *
     * @param canvas the canvas to set up
     * @param ctm current transformation matrix
     */
    void saveAndSetupCanvas(OH_Drawing_Canvas *canvas, drawing::Matrix ctm) {
        OH_Drawing_CanvasSave(canvas);
        cTM_ = mMatrix.Concat(mTransform);
        OH_Drawing_CanvasConcatMatrix(canvas, cTM_.get());
        cTM_ = cTM_.Concat(ctm);
        mInvCTM = cTM_.Invert();
    }

private:
    SvgMarkerAttribute markerAttribute_;
    drawing::Matrix markerTransform;
    drawing::Matrix mMatrix;
    drawing::Matrix mTransform;
    std::optional<drawing::Matrix> mInvCTM;
};

} // namespace svg
} // namespace rnoh
