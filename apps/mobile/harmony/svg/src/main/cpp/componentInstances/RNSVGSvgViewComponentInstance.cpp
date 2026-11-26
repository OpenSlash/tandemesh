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

#include "RNSVGSvgViewComponentInstance.h"
#include "utils/DynamicUtils.h"
#include <glog/logging.h>
#include "RNSVGGroupComponentInstance.h"

namespace rnoh {
namespace svg {

RNSVGSvgViewComponentInstance::RNSVGSvgViewComponentInstance(Context context)
    : CppComponentInstance(std::move(context)) {
    SetSvgNode(m_svgSvg);
    getLocalRootArkUINode().SetSvgNode(GetSvgNode());
    GetSvgNode()->SetContext(std::make_shared<SvgContext>());
    SvgViewManager::getInstance().setSvgView(CppComponentInstance::getTag(), m_svgSvg);
}

RNSVGSvgViewComponentInstance::~RNSVGSvgViewComponentInstance() {
    SvgViewManager::getInstance().onDropView(CppComponentInstance::getTag());
    noSvgComponentIndex = 0;
}

void RNSVGSvgViewComponentInstance::onFinalizeUpdates() {
    ComponentInstance::onFinalizeUpdates();
    std::string bbWidthStr = DynamicUtils::DynamicToString(m_props->bbWidth);
    std::string bbHeightStr = DynamicUtils::DynamicToString(m_props->bbHeight);
    m_svgSvg->SetScale(m_layoutMetrics.pointScaleFactor);
    auto tintColor = getColorFromDynamic(m_props->rawProps);
    if (tintColor.has_value()) {
        m_svgSvg->GetContext()->SetSvgColor(Color::FromString(tintColor.value()));
    }
    m_svgSvg->SetVbX(m_props->minX);
    m_svgSvg->SetVbY(m_props->minY);
    m_svgSvg->SetVbWidth(m_props->vbWidth);
    m_svgSvg->SetVbHeight(m_props->vbHeight);
    m_svgSvg->SetX(0);
    m_svgSvg->SetY(0);
    m_svgSvg->SetWidth(bbWidthStr);
    m_svgSvg->SetHeight(bbHeightStr);
    m_svgSvg->SetAlign(m_props->align);
    m_svgSvg->SetMeetOrSlice(m_props->meetOrSlice);
    getLocalRootArkUINode().markDirty();
}

void RNSVGSvgViewComponentInstance::onDrawForeignImage(ForeignProps foreignProps) {
    if (foreignProps.foreignPixelMap) {
        DLOG(INFO) << "[svgForeignNode] RNSVGSvgViewComponentInstance OH_PixelmapNative is not null, position:{ x:" << foreignProps.x.Value()
                   << ",y:" << foreignProps.y.Value() << "},width:" << foreignProps.width.Value() << ";height:" << foreignProps.height.Value();
        m_svgArkUINode.SetForeignObject(foreignProps);
        m_svgArkUINode.markDirty();
    } else {
        DLOG(INFO) << "RNSVGSvgViewComponentInstance OH_PixelmapNative is null";
    }
}
void RNSVGSvgViewComponentInstance::onChildInserted(ComponentInstance::Shared const &childComponentInstance,
                                                    std::size_t index) {
    CppComponentInstance::onChildInserted(childComponentInstance, index);
    if (childComponentInstance->getComponentName() == "RNSVGGroup") {
        auto childInstance = childComponentInstance->getChildren();
        for (ComponentInstance::Shared c : childInstance) {
            if (c->getComponentName().find("SVG") == std::string::npos) {
                NativeNodeApi::getInstance()->insertChildAt(m_svgArkUINode.getArkUINodeHandle(),
                                                            c->getLocalRootArkUINode().getArkUINodeHandle(), noSvgComponentIndex);
                noSvgComponentIndex++;
            }
            if (c->getComponentName() == "RNSVGForeignObject") {
               auto m_foreignComponentInstance =
                        std::dynamic_pointer_cast<RNSVGForeignObjectComponentInstance>(c);
                    if (m_foreignComponentInstance) {
                        auto groupInstance =
                            std::dynamic_pointer_cast<RNSVGGroupComponentInstance>(c->getParent().lock());
                        m_svgArkUINode.SetGroupNode(groupInstance->getNode());
                        m_svgArkUINode.AddChild(m_foreignComponentInstance->getLocalRootArkUINode());
                        m_foreignComponentInstance->getLocalRootArkUINode().SetForeignNodeDelegate(this);
                    }
            }
            auto groupChildInstance = c->getChildren();
            for (ComponentInstance::Shared c1 : groupChildInstance) {
                if (c1->getComponentName() == "RNSVGForeignObject") {
                    auto m_foreignComponentInstance =
                        std::dynamic_pointer_cast<RNSVGForeignObjectComponentInstance>(c1);
                    if (m_foreignComponentInstance) {
                        auto groupInstance =
                            std::dynamic_pointer_cast<RNSVGGroupComponentInstance>(c1->getParent().lock());
                        m_svgArkUINode.SetGroupNode(groupInstance->getNode());
                        m_svgArkUINode.AddChild(m_foreignComponentInstance->getLocalRootArkUINode());
                        m_foreignComponentInstance->getLocalRootArkUINode().SetForeignNodeDelegate(this);
                    }
                }
            }
        }
    }


    OnChildInsertCommon(std::dynamic_pointer_cast<SvgHost>(childComponentInstance));
}

void RNSVGSvgViewComponentInstance::onChildRemoved(ComponentInstance::Shared const &childComponentInstance) {
    CppComponentInstance::onChildRemoved(childComponentInstance);
    OnChildRemoveCommon(std::dynamic_pointer_cast<SvgHost>(childComponentInstance));
}

SvgArkUINode &RNSVGSvgViewComponentInstance::getLocalRootArkUINode() { return m_svgArkUINode; }

std::optional<std::string> RNSVGSvgViewComponentInstance::getColorFromDynamic(folly::dynamic value) {
    if (value.isNull() || !value.isObject()) {
        return std::nullopt;
    }
    auto rawPropsColor = (value.count("color") > 0) ? std::optional(value["color"].asString()) : std::nullopt;
    return rawPropsColor;
}

bool RNSVGSvgViewComponentInstance::canHandleTouch() const {
    if (m_props != nullptr) {
        auto props = std::dynamic_pointer_cast<const facebook::react::RNSVGSvgViewProps>(m_props);
        return props->pointerEvents == "auto" || props->pointerEvents == "box-only" || props->pointerEvents.size() == 0;
    }
    return true;
};

bool RNSVGSvgViewComponentInstance::canChildrenHandleTouch() const {
    if (m_props != nullptr) {
        auto props = std::dynamic_pointer_cast<const facebook::react::RNSVGSvgViewProps>(m_props);
        return props->pointerEvents == "auto" || props->pointerEvents == "box-none" || props->pointerEvents.size() == 0;
    }
    return true;
};

} // namespace svg
} // namespace rnoh
