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

#include <ReactCommon/TurboModule.h>
#include <react/bridging/Bridging.h>
#include "RNOH/ArkTSTurboModule.h"
#include "SvgViewManager.h"
#include <optional>

using namespace facebook;

namespace rnoh {
namespace svg {

class JSI_EXPORT RNSVGSvgViewModule : public ArkTSTurboModule {
    using Object = std::optional<jsi::Object>;
    using Int32_t = std::optional<int32_t>;
    using Callback = std::optional<std::function<void(std::string)>>;

public:
    RNSVGSvgViewModule(const ArkTSTurboModule::Context ctx, const std::string name);

    void toDataURL(jsi::Runtime &rt, Int32_t tag, Object object, Callback &&callback);
};

} // namespace svg
} // namespace rnoh
