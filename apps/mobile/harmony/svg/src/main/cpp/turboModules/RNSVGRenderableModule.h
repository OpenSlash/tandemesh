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

using namespace facebook;

namespace rnoh {
namespace svg {

class JSI_EXPORT RNSVGRenderableModule : public ArkTSTurboModule {
public:
    RNSVGRenderableModule(const ArkTSTurboModule::Context ctx, const std::string name);
    bool isPointInFill(jsi::Runtime &rt, int32_t tag, std::optional<jsi::Object> options);
    bool isPointInStroke(jsi::Runtime &rt, int32_t tag, std::optional<jsi::Object> options);
    double getTotalLength(jsi::Runtime &rt, int32_t tag);
    jsi::Object getPointAtLength(jsi::Runtime &rt, int32_t tag, std::optional<jsi::Object> options);
    jsi::Object getBBox(jsi::Runtime &rt, int32_t tag, std::optional<jsi::Object> options);
    jsi::Object getCTM(jsi::Runtime &rt, int32_t tag);
    jsi::Object getScreenCTM(jsi::Runtime &rt, int32_t tag);
    jsi::Value getRawResource(jsi::Runtime &rt, jsi::String name);
};

} // namespace svg
} // namespace rnoh
