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

#ifndef HARMONY_SVGFOREIGNOBJECTNODE_H
#define HARMONY_SVGFOREIGNOBJECTNODE_H
#include "RNOH/arkui/ArkUINode.h"
#include "RNOH/arkui/StackNode.h"
#include "SvgForeignObjectNodeDelegate.h"
#include "properties/Dimension.h"

namespace rnoh {
namespace svg {
class SvgForeignObjectNode : public ArkUINode {
public:
    SvgForeignObjectNode();
    ~SvgForeignObjectNode() override;
    void onNodeEvent(ArkUI_NodeEventType eventType, EventArgs &eventArgs) override;
    StackNode &getSnapNode() { return mStackNode; }
    void insertChild(ArkUINode &child, std::size_t index);
    void reDrawForeignObject();
    void SetSnapPosition(Dimension x, Dimension y);
    void SetSnapWidth(Dimension width);
    void SetSnapHeight(Dimension height);
    void SetPointScaleFactor(float pointScaleFactor) {
        pointScaleFactor_ = pointScaleFactor;
    }
    void SetTransform(std::vector<double> transform) {
        transform_ = transform;
    }
    void setNodeSize(float w, float h){
        mStackNode.setSize({w, h});
    }
    OH_PixelmapNative *GetNodePixelMap();
    void SetForeignNodeDelegate(SvgForeignObjectNodeDelegate *delegate) { m_NodeDelegate = delegate; };
    void SetGeneratedPixelMap(bool isNeed) {
        _isGeneratedPixelMap = isNeed;
    }
    
    void SetClipPath(std::string path,int clipRule) {
        _path = path;
        _clipRule = clipRule;
    }
     void SetMask(std::string mask) {
        _mask = mask;
    }
    
private:
    StackNode mStackNode;
    SvgForeignObjectNodeDelegate *m_NodeDelegate;
    Dimension _width{0};
    Dimension _height{0};
    Dimension _positionX{0};
    Dimension _positionY{0};
    std::string _path{""};
    std::string _mask{""};
    std::vector<double> transform_;
    int _clipRule{0};
    float pointScaleFactor_{0};
    bool _isGeneratedPixelMap{false}; //防止快照生成多次，导致性能影响
	bool _isGeneratedPixelMapByNodeEvent{false}; // 防止重复进行矩阵变换
    void *high_lib_handle = NULL;
    ArkUI_SnapshotOptions* (*createSnapshotOptionsMethod)();
    int32_t (*snapshotOptionsSetScaleMethod)(ArkUI_SnapshotOptions* snapshotOptions, float scale);
    int32_t (*getNodeSnapshotMethod)(ArkUI_NodeHandle node, ArkUI_SnapshotOptions* snapshotOptions,
    OH_PixelmapNative** pixelmap);
};

} // namespace svg
} // namespace rnoh
#endif // HARMONY_SVGFOREIGNOBJECTNODE_H
