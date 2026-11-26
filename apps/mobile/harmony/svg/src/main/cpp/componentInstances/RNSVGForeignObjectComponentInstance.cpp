/*
 * Copyright (c) 2025 Huawei Device Co., Ltd. All rights reserved
 * Use of this source code is governed by a MIT license that can be
 * found in the LICENSE file.
 *
 * This file incorporates from the OpenHarmony project, licensed under
 * the Apache License, Version 2.0. Specifically:
 * - [OpenHarmony/arkui_ace_engine] (https://gitee.com/openharmony/arkui_ace_engine)
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License. You may obtain a copy
 * of the License at:
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 */

#include "RNSVGForeignObjectComponentInstance.h"
namespace rnoh {
namespace svg {

RNSVGForeignObjectComponentInstance::RNSVGForeignObjectComponentInstance(Context context)
    : CppComponentInstance(std::move(context)) {
    DLOG(INFO) << "RNSVGForeignObjectComponentInstance";
    NativeNodeApi::getInstance()->registerNodeEvent(mForeignStackNode.getArkUINodeHandle(), NODE_EVENT_ON_AREA_CHANGE,
                                                    NODE_EVENT_ON_AREA_CHANGE, this);
    mForeignStackNode.insertChild(node, 0);
}


RNSVGForeignObjectComponentInstance::~RNSVGForeignObjectComponentInstance() {
    NativeNodeApi::getInstance()->unregisterNodeEvent(mForeignStackNode.getArkUINodeHandle(),
                                                      NODE_EVENT_ON_AREA_CHANGE);
}

void RNSVGForeignObjectComponentInstance::onFinalizeUpdates() {
    ComponentInstance::onFinalizeUpdates();
    if (m_props) {
        mForeignStackNode.SetPointScaleFactor(getLayoutMetrics().pointScaleFactor);
        mForeignStackNode.SetSnapPosition(propsConversionValue(m_props->x), propsConversionValue(m_props->y));
        mForeignStackNode.SetSnapWidth(propsConversionValue(m_props->width));
        mForeignStackNode.SetSnapHeight(propsConversionValue(m_props->height));

        mForeignStackNode.SetClipPath(m_props->clipPath, m_props->clipRule);
        mForeignStackNode.SetMask(m_props->mask);
        mForeignStackNode.SetTransform(m_props->matrix);
        auto childs = getChildren();
        if (childs.size() > 0) {
            for (ComponentInstance::Shared c : childs) {
                if ((m_props->opacity > 0 && m_props->opacity != 1)) {
                    setOpacity(c->getLocalRootArkUINode(), m_props->opacity);
                }
            }
        }
        mForeignStackNode.SetGeneratedPixelMap(true);
    }
}

Dimension RNSVGForeignObjectComponentInstance::propsConversionValue(const folly::dynamic &d) {
    if (d.isNull()) {
        return Dimension(0, DimensionUnit::INVALID);
    }
    return StringUtils::StringToDimension(d.asString(), true);
}

void RNSVGForeignObjectComponentInstance::setOpacity(ArkUINode &node, float op) {
    if (op != 1) {
        node.setOpacity(m_props->opacity);
    }
}

void OnDrawCompleted(void *userData) {
    RNSVGForeignObjectComponentInstance *ctx = static_cast<RNSVGForeignObjectComponentInstance *>(userData);
    if (ctx != nullptr) {
        ctx->getLocalRootArkUINode().reDrawForeignObject();
    }
}

void RNSVGForeignObjectComponentInstance::onChildInserted(ComponentInstance::Shared const &childComponentInstance,
                                                          std::size_t index) {
    CppComponentInstance::onChildInserted(childComponentInstance, index);
    float width = childComponentInstance->getLayoutMetrics().frame.size.width;
    float height = childComponentInstance->getLayoutMetrics().frame.size.height;
    high_lib_handle = dlopen("libace_ndk.z.so", RTLD_LAZY);
    if (high_lib_handle == NULL) {
        DLOG(INFO) << "[svgForeignNode] libace_ndk.z.so not existed";
        dlclose(high_lib_handle);
    } else {
        registerDrawCallbackOnNodeHandleMethod =
            (int32_t(*)(ArkUI_NodeHandle node, void *userData, void (*onDrawCompleted)(void *userData)))dlsym(
                high_lib_handle, "OH_ArkUI_RegisterDrawCallbackOnNodeHandle");
        bool isRegisterDrawCallbackOnNodeHandleMethodExisted = registerDrawCallbackOnNodeHandleMethod != NULL;
        if (isRegisterDrawCallbackOnNodeHandleMethodExisted) {
            DLOG(INFO) << "[svgForeignNode] method existed";
            registerDrawCallbackOnNodeHandleMethod(
                childComponentInstance->getLocalRootArkUINode().getArkUINodeHandle(), this, OnDrawCompleted);
        }
        dlclose(high_lib_handle);
    }
   
    mForeignStackNode.setNodeSize(width, height);
    node.insertChild(childComponentInstance->getLocalRootArkUINode(), index);
}

void RNSVGForeignObjectComponentInstance::onChildRemoved(ComponentInstance::Shared const &childComponentInstance) {
    CppComponentInstance::onChildRemoved(childComponentInstance);
    high_lib_handle = dlopen("libace_ndk.z.so", RTLD_LAZY);
    if (high_lib_handle == NULL) {
        DLOG(INFO) << "[svgForeignNode] libace_ndk.z.so not existed";
        dlclose(high_lib_handle);
    } else {
        unregisterDrawCallbackOnNodeHandleMethod =
            (int32_t(*)(ArkUI_NodeHandle node))dlsym(high_lib_handle, "OH_ArkUI_UnregisterDrawCallbackOnNodeHandle");
        bool isUnRegisterDrawCallbackOnNodeHandleMethodExisted = registerDrawCallbackOnNodeHandleMethod != NULL;
        if (isUnRegisterDrawCallbackOnNodeHandleMethodExisted) {
            DLOG(INFO) << "[svgForeignNode] method existed";
            unregisterDrawCallbackOnNodeHandleMethod(childComponentInstance->getLocalRootArkUINode().getArkUINodeHandle());
        }
        dlclose(high_lib_handle);
    }
    
}

SvgForeignObjectNode &RNSVGForeignObjectComponentInstance::getLocalRootArkUINode() { return mForeignStackNode; }


} // namespace svg
}