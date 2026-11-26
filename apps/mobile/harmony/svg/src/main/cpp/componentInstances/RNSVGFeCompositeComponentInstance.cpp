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

#include "RNSVGFeCompositeComponentInstance.h"
#include "utils/DynamicUtils.h"
#include <glog/logging.h>

namespace rnoh {
namespace svg {

RNSVGFeCompositeComponentInstance::RNSVGFeCompositeComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)) {}

void RNSVGFeCompositeComponentInstance::onPropsChanged(SharedConcreteProps const &props) {
    if (props) {
        x_ = DynamicUtils::DynamicToString(props->x);
        y_ = DynamicUtils::DynamicToString(props->y);
        width_ = DynamicUtils::DynamicToString(props->width);
        height_ = DynamicUtils::DynamicToString(props->height);
        in_ = props->in1;
        in2_ = props->in2;
        operator_ = facebook::react::toString(props->operator1);
        result_ = props->result;
        k1_ = std::to_string(props->k1);
        k2_ = std::to_string(props->k2);
        k3_ = std::to_string(props->k3);
        k4_ = std::to_string(props->k4);
    }
}

void RNSVGFeCompositeComponentInstance::UpdateElementProps() {}

void RNSVGFeCompositeComponentInstance::extractAttributes() {}

} // namespace svg
} // namespace rnoh
