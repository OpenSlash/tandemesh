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
#include "SvgFilter.h"
#include "utils/FilterManager.h"
#include <react/renderer/components/react_native_svg/ShadowNodes.h>

namespace rnoh {
namespace svg {


class RNSVGFilterComponentInstance : public RNSVGBaseComponentInstance<facebook::react::RNSVGFilterShadowNode> {
public:
    RNSVGFilterComponentInstance(Context context);

    void onPropsChanged(SharedConcreteProps const &props) override;
    void UpdateElementProps() override;

private:
    std::shared_ptr<SvgFilter> m_svgFilter = std::make_shared<SvgFilter>();
    std::string filterId_;
    
    void parseFilterId();
    void collectFilterEffectsInOrder();
    bool collectFeDropShadowInfo();
    void collectSingleFeColorMatrix(std::shared_ptr<ComponentInstance> childInstance);
    void collectSingleFeFlood(std::shared_ptr<ComponentInstance> childInstance);
    void collectSingleFeOffset(std::shared_ptr<ComponentInstance> childInstance);
    void collectSingleFeComposite(std::shared_ptr<ComponentInstance> childInstance);
    void collectSingleFeGaussianBlur(std::shared_ptr<ComponentInstance> childInstance);
    void collectSingleFeBlend(std::shared_ptr<ComponentInstance> childInstance);
    std::vector<float> generateColorMatrix(const std::string& type, const std::vector<float>& values);
    std::vector<float> convertBlendModeToValues(const std::string& mode);
    std::vector<float> convertCompositeOperatorToValues(const std::string& operatorType, 
                                                        const std::string& k1, 
                                                        const std::string& k2, 
                                                        const std::string& k3, 
                                                        const std::string& k4);
};

} // namespace svg
} // namespace rnoh
