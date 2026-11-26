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

#include "RNSVGImageComponentInstance.h"
#include "turboModules/RNSVGImageModule.h"
#include "utils/DynamicUtils.h"

namespace rnoh {
namespace svg {

RNSVGImageComponentInstance::RNSVGImageComponentInstance(Context context)
    : RNSVGBaseComponentInstance(std::move(context)),
      ImageSourceResolver::ImageSourceUpdateListener(
          (std::dynamic_pointer_cast<RNSVGImageModule>(m_deps->rnInstance.lock()->getTurboModule("RNSVGImageModule")))
              ->getImageSourceResolver()) {
    SetSvgNode(m_svgImage);
}

void RNSVGImageComponentInstance::UpdateElementProps() {
    LOG(INFO) << "XHL === RNSVGImageComponentInstance::UpdateElementProps call";
    if (m_deps != nullptr && !m_deps->rnInstance.expired()) {
        auto rnInstance = m_deps->rnInstance.lock();
        if (rnInstance != nullptr) {
            auto nativeResourceManager = rnInstance->getNativeResourceManager();
            m_svgImage->setNativeResourceManager(nativeResourceManager);
        }
    }
    m_svgImage->UpdateCommonProps(m_props);
    m_svgImage->SetX(DynamicUtils::DynamicToString(m_props->x));
    m_svgImage->SetY(DynamicUtils::DynamicToString(m_props->y));
    m_svgImage->SetWidth(DynamicUtils::DynamicToString(m_props->width));
    m_svgImage->SetHeight(DynamicUtils::DynamicToString(m_props->height));
    m_svgImage->SetAlign(m_props->align);
    m_svgImage->SetMeetOrSlice(m_props->meetOrSlice);
    m_svgImage->SetImageSource(m_src);
    LOG(INFO) << "XHL === RNSVGImageComponentInstance::UpdateElementProps end";
}

void RNSVGImageComponentInstance::onPropsChanged(SharedConcreteProps const &props) {
    if (!m_props || m_props->src != props->src) {
        if (!props->src.uri.empty() && props->src.uri.find("http", 0) != 0) {
            // 非网络资源
            m_src.uri = props->src.uri;
            return;
        }
        std::string cacheDir = FindLocalCacheByUri(props->src.uri);
        if (!cacheDir.empty()) {
            DLOG(INFO) << "[SVGImage] hit cache: " << cacheDir;
            m_src.uri = cacheDir;
        }
    }
};

std::string RNSVGImageComponentInstance::FindLocalCacheByUri(std::string const &uri) {
    // 非网络资源，直接返回uri
    if (uri.find("http", 0) != 0) {
        return uri;
    }

    auto rnInstance = m_deps->rnInstance.lock();
    if (!rnInstance) {
        return uri;
    }

    auto turboModule = rnInstance->getTurboModule("RNSVGImageModule");
    if (!turboModule) {
        return uri;
    }

    auto arkTsTurboModule = std::dynamic_pointer_cast<RNSVGImageModule>(turboModule);
    if (!arkTsTurboModule || !arkTsTurboModule->getImageSourceResolver()) {
        LOG(ERROR) << "[SVGImage] FastImageSourceResolver is unavailable!";
        return uri;
    }

    // 查找缓存目录
    auto basePtr = shared_from_this();
    auto thisPtr = std::static_pointer_cast<RNSVGImageComponentInstance>(basePtr);
    auto cacheDir = arkTsTurboModule->getImageSourceResolver()->resolveImageSources(thisPtr, uri);
    
    if (!cacheDir.empty()) {
        return cacheDir; // 如果找到缓存文件，返回缓存路径
    }
    // 否则开始下载
    arkTsTurboModule->downloadImage(uri, {});
    return ""; // 返回空字符串，表示下载未完成或缓存未命中
}

void RNSVGImageComponentInstance::onImageSourceCacheUpdate(std::string imageUri) {
    LOG(INFO) << "[SVGImage] onImageSourceCacheUpdate, uri: " << imageUri;
    m_src.uri = imageUri;
    onFinalizeUpdates();
};

void RNSVGImageComponentInstance::onImageSourceCacheDownloadFileFail() {
    LOG(ERROR) << "[SVGImage] ImageSource download fail!";
};

} // namespace svg
} // namespace rnoh
