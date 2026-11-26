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

#include "RNOH/CppComponentInstance.h"
#include <folly/dynamic.h>
#include "SvgArkUINode.h"
#include <react/renderer/components/react_native_svg/ShadowNodes.h>
#include "SvgViewManager.h"
#include "SvgSvg.h"
#include "RNSVGForeignObjectComponentInstance.h"

namespace rnoh {
namespace svg {
class RNSVGGroupComponentInstance;
} // namespace svg
} // namespace rnoh


namespace rnoh {
namespace svg {

class RNSVGSvgViewComponentInstance : public CppComponentInstance<facebook::react::RNSVGSvgViewShadowNode>,
                                      public SvgHost , public SvgForeignObjectNodeDelegate {
public:
    explicit RNSVGSvgViewComponentInstance(Context context);
    ~RNSVGSvgViewComponentInstance();

    void onFinalizeUpdates() override;
    void onDrawForeignImage(ForeignProps foreignProps) override;
    // get SvgNode from childComponentInstance and set it to root_
    void onChildInserted(ComponentInstance::Shared const &childComponentInstance, std::size_t index) override;

    void onChildRemoved(ComponentInstance::Shared const &childComponentInstance) override;

    SvgArkUINode &getLocalRootArkUINode() override;

    // since we can't get color correctly from props
    std::optional<std::string> getColorFromDynamic(folly::dynamic value);

    bool canHandleTouch() const override;

    bool canChildrenHandleTouch() const override;

private:
    SvgArkUINode m_svgArkUINode;
    std::shared_ptr<SvgSvg> m_svgSvg = std::make_shared<SvgSvg>();
    int noSvgComponentIndex {0}; // 非svg添加的index(适配js侧view包裹会拆解子元素成平级)
};

} // namespace svg
} // namespace rnoh