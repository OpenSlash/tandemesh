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

#include "SvgGraphic.h"
#include <native_drawing/drawing_point.h>
#include <native_drawing/drawing_canvas.h>
#include <native_drawing/drawing_path.h>
#include "SvgSymbol.h"

namespace rnoh {
namespace svg {

    class SvgUse : public SvgGraphic {
    public:
        SvgUse() = default;
        ~SvgUse() override = default;
        float x;
        float y;
        float width;
        float height;
        float opacity;
        std::string href;

        void OnDraw(OH_Drawing_Canvas *canvas) override;
    
        drawing::Path AsPath() override;
    private:
        class AttributeScope {
            public:
                explicit AttributeScope(const std::shared_ptr<SvgNode>& node);
                ~AttributeScope();

            private:
                SvgBaseAttribute attributes_;
                std::shared_ptr<SvgNode> node_;
            };
                // SvgAttributes useAttr_;
        };

} // namespace svg
} // namespace rnoh