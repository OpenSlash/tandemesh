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
#include <dlfcn.h>
#include "SvgForeignObjectNode.h"
#include "RNOH/arkui/NativeNodeApi.h"

namespace rnoh {
namespace svg {
SvgForeignObjectNode::SvgForeignObjectNode()
    : ArkUINode(NativeNodeApi::getInstance()->createNode(ArkUI_NodeType::ARKUI_NODE_STACK)) {
    mStackNode.setVisibility(ArkUI_Visibility::ARKUI_VISIBILITY_HIDDEN);
    mStackNode.setClip(true);
    maybeThrow(NativeNodeApi::getInstance()->addChild(m_nodeHandle, mStackNode.getArkUINodeHandle()));
}

SvgForeignObjectNode::~SvgForeignObjectNode() {
    if (m_NodeDelegate) {
        m_NodeDelegate = nullptr;
    }
    _isGeneratedPixelMap = false;
}
void SvgForeignObjectNode::insertChild(ArkUINode &child, std::size_t index) { mStackNode.insertChild(child, index); }

void SvgForeignObjectNode::SetSnapHeight(Dimension height) { _height = height; }

void SvgForeignObjectNode::SetSnapWidth(Dimension width) { _width = width; }

void SvgForeignObjectNode::SetSnapPosition(Dimension x, Dimension y) {
    _positionX = x;
    _positionY = y;
}

void SvgForeignObjectNode::onNodeEvent(ArkUI_NodeEventType eventType, EventArgs &eventArgs) {
    if (eventType == ArkUI_NodeEventType::NODE_EVENT_ON_AREA_CHANGE) {
        if (m_NodeDelegate && _isGeneratedPixelMap) {
            OH_PixelmapNative *pixelMap = GetNodePixelMap();
            if (!pixelMap) {
                LOG(ERROR) << "[svgForeignNode] get node snapshot pixelMap is null";
                return;
            }
            _isGeneratedPixelMap = false;
			_isGeneratedPixelMapByNodeEvent = true;
            ForeignProps foreignProps = {pixelMap, _width, _height, _positionX, 
                _positionY, _path, _clipRule, _mask, pointScaleFactor_, transform_};
            m_NodeDelegate->onDrawForeignImage(foreignProps);
        }
    }
}

void SvgForeignObjectNode::reDrawForeignObject() {
    if (m_NodeDelegate && !_isGeneratedPixelMapByNodeEvent) {
        OH_PixelmapNative *pixelMap = GetNodePixelMap();
        if (!pixelMap) {
            LOG(ERROR) << "[svgForeignNode] get node snapshot pixelMap is null";
            return;
        }
        _isGeneratedPixelMap = false;
        ForeignProps foreignProps = {pixelMap, _width,    _height, _positionX,        _positionY,
                                     _path,    _clipRule, _mask,   pointScaleFactor_, transform_};
        m_NodeDelegate->onDrawForeignImage(foreignProps);
    } else {
        _isGeneratedPixelMapByNodeEvent = false;
    }
}

OH_PixelmapNative *SvgForeignObjectNode::GetNodePixelMap() {
    high_lib_handle = dlopen("libace_ndk.z.so", RTLD_LAZY);
    if (high_lib_handle == NULL) {
        DLOG(INFO) << "[svgForeignNode] libace_ndk.z.so not existed";
        dlclose(high_lib_handle);
        return nullptr;
    }
    createSnapshotOptionsMethod = (ArkUI_SnapshotOptions*(*)())dlsym(high_lib_handle, "OH_ArkUI_CreateSnapshotOptions");
    snapshotOptionsSetScaleMethod = (int32_t(*)(ArkUI_SnapshotOptions*, float))dlsym(high_lib_handle, "OH_ArkUI_SnapshotOptions_SetScale");
    getNodeSnapshotMethod = (int32_t(*)(ArkUI_NodeHandle, ArkUI_SnapshotOptions*, OH_PixelmapNative** ))dlsym(high_lib_handle, "OH_ArkUI_GetNodeSnapshot");
    
    bool isCreateSnapshotOptionsMethodExisted = createSnapshotOptionsMethod != NULL;
    bool isSnapshotOptionsSetScaleMethodExisted = snapshotOptionsSetScaleMethod != NULL;
    bool isGetNodeSnapshotMethodExisted = getNodeSnapshotMethod != NULL;
    if (isCreateSnapshotOptionsMethodExisted && isSnapshotOptionsSetScaleMethodExisted && isGetNodeSnapshotMethodExisted) {
        DLOG(INFO) << "[svgForeignNode] method existed";
        OH_PixelmapNative *pixelMap;
        ArkUI_SnapshotOptions *options = createSnapshotOptionsMethod();
        snapshotOptionsSetScaleMethod(options, 1);
        int32_t code = getNodeSnapshotMethod(mStackNode.getArkUINodeHandle(), options, &pixelMap);
        if (code == ARKUI_ERROR_CODE_NO_ERROR) {
            dlclose(high_lib_handle);
            return pixelMap;
        }
    }
    dlclose(high_lib_handle);
    return nullptr;
}

} // namespace svg
} // namespace rnoh
