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


namespace rnoh {
namespace svg {

class RNSVGFeCompositeComponentInstance : public RNSVGBaseComponentInstance<facebook::react::RNSVGFeCompositeShadowNode> {
public:
    using SharedConcreteProps = std::shared_ptr<const facebook::react::RNSVGFeCompositeProps>;

    RNSVGFeCompositeComponentInstance(Context context);

    void onPropsChanged(SharedConcreteProps const &props) override;
    void UpdateElementProps() override;
    
    std::string getIn() const { return in_; }
    std::string getIn2() const { return in2_; }
    std::string getOperator() const { return operator_; }
    std::string getResult() const { return result_; }
    std::string getK1() const { return k1_; }
    std::string getK2() const { return k2_; }
    std::string getK3() const { return k3_; }
    std::string getK4() const { return k4_; }
    std::string getX() const { return x_; }
    std::string getY() const { return y_; }
    std::string getWidth() const { return width_; }
    std::string getHeight() const { return height_; }

private:
    std::string x_ = "0";
    std::string y_ = "0";
    std::string width_ = "0";
    std::string height_ = "0";
    std::string in_ = "SourceGraphic";
    std::string in2_ = "";
    std::string operator_ = "over";
    std::string result_ = "";
    std::string k1_ = "0";
    std::string k2_ = "0";
    std::string k3_ = "0";
    std::string k4_ = "0";
    void extractAttributes();
};

} // namespace svg
} // namespace rnoh
