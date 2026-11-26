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

#ifndef RNSVG_FOREIGN_OBJECT_COMPONENT_INSTANCE_H
#define RNSVG_FOREIGN_OBJECT_COMPONENT_INSTANCE_H
#include "RNOH/CppComponentInstance.h"
#include "SvgForeignObjectNode.h"
#include <react/renderer/components/react_native_svg/ShadowNodes.h>
#include "RNOH/arkui/ColumnNode.h"
#include "RNOH/arkui/NativeNodeApi.h"
#include "utils/DynamicUtils.h"
#include "properties/Dimension.h"
#include "utils/StringUtils.h"
namespace rnoh {
namespace svg {
class RNSVGForeignObjectComponentInstance : public CppComponentInstance<facebook::react::RNSVGForeignObjectShadowNode> {
public:
    RNSVGForeignObjectComponentInstance(Context context);
    ~RNSVGForeignObjectComponentInstance();

    void onFinalizeUpdates() override;
    void onChildInserted(ComponentInstance::Shared const &childComponentInstance, std::size_t index) override;

    void onChildRemoved(ComponentInstance::Shared const &childComponentInstance) override;

    SvgForeignObjectNode &getLocalRootArkUINode() override;
    
    void setOpacity(ArkUINode &node, float op);
    
    Dimension propsConversionValue(const folly::dynamic& d);
private:
    SvgForeignObjectNode mForeignStackNode;
    ColumnNode node;
    void *high_lib_handle = NULL;
    int32_t (*registerDrawCallbackOnNodeHandleMethod)(ArkUI_NodeHandle node,
                                                      void* userData, void (*onDrawCompleted)(void* userData));
    int32_t (*unregisterDrawCallbackOnNodeHandleMethod)(ArkUI_NodeHandle node);
};
} // namespace svg
} // namespace rnoh
#endif // RNSVG_FOREIGN_OBJECT_COMPONENT_INSTANCE_H
