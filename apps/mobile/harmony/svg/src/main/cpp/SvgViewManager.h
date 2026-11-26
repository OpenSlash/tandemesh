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

#include <future>
#include "SvgSvg.h"
#include <native_drawing/drawing_bitmap.h>
#include <mutex>

namespace rnoh {
namespace svg {

class SvgViewManager {
public:
    static SvgViewManager &getInstance() {
        // 单例模式
        static SvgViewManager instance;
        return instance;
    }
    void setSvgView(int tag, std::weak_ptr<SvgSvg> svg);

    std::weak_ptr<SvgSvg> getSvgViewByTag(int tag);

    int32_t getHitNode(int32_t tag, double x, double y);

    void onDropView(int tag);

private:
    SvgViewManager() {}
    SvgViewManager(const SvgViewManager &) = delete;
    SvgViewManager &operator=(const SvgViewManager &) = delete;
    ~SvgViewManager();
    std::map<int, std::weak_ptr<SvgSvg>> mTagToSvgViewMap;
    std::mutex mTagtoSvgViewMapLock;
};
} // namespace svg
} // namespace rnoh
