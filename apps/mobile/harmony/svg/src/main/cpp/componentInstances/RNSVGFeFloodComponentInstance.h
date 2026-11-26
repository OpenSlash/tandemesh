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

#include "RNSVGBaseComponentInstance.h"
#include <react/renderer/components/react_native_svg/ShadowNodes.h>
#include <react/renderer/components/react_native_svg/Props.h>

namespace rnoh {
namespace svg {

class RNSVGFeFloodComponentInstance : public RNSVGBaseComponentInstance<facebook::react::RNSVGFeFloodShadowNode> {
public:
    using SharedConcreteProps = std::shared_ptr<const facebook::react::RNSVGFeFloodProps>;

    RNSVGFeFloodComponentInstance(Context context);
    virtual ~RNSVGFeFloodComponentInstance() = default;

    void onPropsChanged(SharedConcreteProps const &props) override;
    void UpdateElementProps() override;
    
    std::string getX() const { return x_; }
    std::string getY() const { return y_; }
    std::string getWidth() const { return width_; }
    std::string getHeight() const { return height_; }
    std::string getResult() const { return result_; }
    std::string getIn() const { return in_; }
    std::string getFloodColor() const { return floodColor_; }
    float getFloodOpacity() const { return floodOpacity_; }

private:
    std::string x_ = "0";
    std::string y_ = "0";
    std::string width_ = "0";
    std::string height_ = "0";
    std::string result_ = "";
    std::string in_ = "SourceGraphic";
    std::string floodColor_{};
    float floodOpacity_ = 1.0f;
    
    void extractAttributes();
};

} // namespace svg
} // namespace rnoh
