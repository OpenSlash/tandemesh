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

#include "RNOH/arkui/ArkUINode.h"
#include "SvgForeignObjectNodeDelegate.h"
#include "SvgForeignProps.h"
#include "SvgHost.h"
#include "SvgNode.h"
#include "arkui/native_node.h"

namespace rnoh {
namespace svg {

struct UserCallback {
    std::function<void(ArkUI_NodeCustomEvent *event)> callback;
};

// ArkUINode
class SvgArkUINode : public ArkUINode {
private:
    void OnDraw(ArkUI_NodeCustomEvent *event);
    
    std::weak_ptr<SvgNode> root_;
    ArkUI_NativeNodeAPI_1 *nativeModule_ = nullptr;
    UserCallback *userCallback_ = nullptr;
    std::weak_ptr<SvgNode> _groupNode;
    std::vector<ForeignProps> foreignPropsArray;
    void (*eventReceiver)(ArkUI_NodeCustomEvent *event);
 
public:
    SvgArkUINode();
    ~SvgArkUINode() override;

    void SetSvgNode(const std::weak_ptr<SvgNode> &node) { root_ = node; }
    void SetGroupNode(const std::weak_ptr<SvgNode> &node) { _groupNode = node; }
    void ResetNodeHandle() {}
    void AddChild(ArkUINode &node);
    void SetForeignObject(ForeignProps foreignProps) {
        if (auto groupNode = _groupNode.lock()) {
            groupNode->SetForeignObject(std::move(foreignProps));
        }
    }
};

} // namespace svg
} // namespace rnoh