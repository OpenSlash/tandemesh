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

#include <memory>
#include "SvgNode.h"

namespace rnoh {
namespace svg {

class SvgHost {
 public:
     SvgHost() = default;
     virtual ~SvgHost() = default;
     void SetSvgNode(const std::shared_ptr<SvgNode> &svgNode) { 
        std::lock_guard<std::mutex> lock(mtx);
        m_svgNode = svgNode;
    };
     const std::shared_ptr<SvgNode> &GetSvgNode() const { 
        std::lock_guard<std::mutex> lock(mtx);
        return m_svgNode;
    };

     void OnChildInsertCommon(const std::shared_ptr<SvgHost> &childSvgHost);

     void OnChildRemoveCommon(const std::shared_ptr<SvgHost> &childSvgHost);

 private:
  std::shared_ptr<SvgNode> m_svgNode{nullptr};
  mutable std::mutex mtx;
};

} // namespace svg
} // namespace rnoh