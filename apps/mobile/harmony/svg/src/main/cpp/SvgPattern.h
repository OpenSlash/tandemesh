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

#include <string>
#include "SvgQuote.h"
#include "utils/SvgAttributesParser.h"

namespace rnoh {
namespace svg {

class SvgPattern : public SvgQuote {

public:
    using Float = facebook::react::Float;
    SvgPattern();
    ~SvgPattern() override = default;
    
    void OnDrawTraversedBefore(OH_Drawing_Canvas *canvas) override;
    void OnDrawTraversedAfter(OH_Drawing_Canvas *canvas) override;
    void SaveDefinition(OH_Drawing_Canvas *canvas);
    void ConcatMatrix(OH_Drawing_Canvas *canvas);
    std::shared_ptr<PatternAttr> GetPatternAttr();

    void setPatternX(Dimension x);

    void setPatternY(Dimension y);

    void setHeight(Dimension height);

    void setWidth(Dimension width);

    void setPatternUnits(int patternUnits);

    void setPatternContentUnits(int patternContentUnits);

    void setViewBox(Rect viewBox);

    void setImage(OH_Drawing_Image *image);

    void setmMinX(float mMinX);

    void setmMinY(float mMinUY);

    void setmVbWidth(float mVbWidth);

    void setmVbHeight(float mVbHeight);

    void setmAlign(std::string mAlign);

    void setmMeetOrSlice(int mMeetOrSlice);

    void setPatternTransforms(std::vector<Float> gradientTransforms);
private:
    std::shared_ptr<PatternAttr> patternAttr = std::make_shared<PatternAttr>();
    double scale_ = 3.25010318;
  };

} // namespace svg
} // namespace rnoh