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

#include "RNSVGSvgViewModule.h"

using namespace facebook;
using namespace react;

namespace rnoh {
namespace svg {

using Object = std::optional<jsi::Object>;

void RNSVGSvgViewModule::toDataURL(jsi::Runtime &rt, Int32_t tag, Object object, Callback &&callback) {
    // 获取图片的bitmap并将其转化为base64
    std::string base64;
    auto weakSvgNode = SvgViewManager::getInstance().getSvgViewByTag(*tag);
    if (weakSvgNode.lock()) {
        auto svgRootNode = weakSvgNode.lock();
        OH_Drawing_Bitmap *bitmap = OH_Drawing_BitmapCreate();
        OH_Drawing_BitmapFormat format = {COLOR_FORMAT_BGRA_8888, ALPHA_FORMAT_OPAQUE};
        auto size = svgRootNode->GetSize();
        OH_Drawing_BitmapBuild(bitmap, size.Width(), size.Height(), &format);
        OH_Drawing_Canvas *canvas = OH_Drawing_CanvasCreate();
        OH_Drawing_CanvasBind(canvas, bitmap);
        // 设置SvgNode节点相关信息
        if (svgRootNode->GetContext()) {
            svgRootNode->GetContext()->ClearIdMap();
        }
        svgRootNode->ContextTraversal();
        svgRootNode->InitStyle({});
        svgRootNode->Draw(canvas);
        base64 = StringUtils::bitmapToBase64(bitmap);
        // 销毁bitmap和canvas
        OH_Drawing_CanvasDestroy(canvas);
        // 销毁bitmap对象
        OH_Drawing_BitmapDestroy(bitmap);
    }
    jsInvoker_->invokeAsync([base64, endCallback = std::move(*callback)] { endCallback(base64); });
    return;
}

static jsi::Value __hostFunction_RNSVGSvgViewModule_toDataURL(jsi::Runtime &rt, react::TurboModule &turboModule,
                                                              const jsi::Value *args, size_t count) {
    auto tag = args[0].isNull() || args[0].isUndefined() ? std::nullopt : std::make_optional(args[0].asNumber());
    auto options = args[1].isNull() || args[1].isUndefined() ? std::nullopt : std::make_optional(args[1].asObject(rt));
    auto self = static_cast<RNSVGSvgViewModule *>(&turboModule);
    if (args[2].isObject()) {
        auto callback = std::make_shared<jsi::Function>(std::move(args[2].getObject(rt).getFunction(rt)));
        auto endCallback = [&rt, callback = std::move(callback)](std::string base64) {
            auto base64Value = jsi::String::createFromUtf8(rt, base64);
            callback->call(rt, {std::move(base64Value)});
        };
        static_cast<RNSVGSvgViewModule *>(&turboModule)->toDataURL(rt, tag, std::move(options), std::move(endCallback));
        return jsi::Value::undefined();
    }
    static_cast<RNSVGSvgViewModule *>(&turboModule)
        ->toDataURL(rt, tag, std::move(options), [self, &rt](std::string base64) {});
    return jsi::Value::undefined();
}

RNSVGSvgViewModule::RNSVGSvgViewModule(const ArkTSTurboModule::Context ctx, const std::string name)
    : ArkTSTurboModule(ctx, name) {
    { methodMap_["toDataURL"] = MethodMetadata{1, __hostFunction_RNSVGSvgViewModule_toDataURL}; };
}

} // namespace svg
} // namespace rnoh
