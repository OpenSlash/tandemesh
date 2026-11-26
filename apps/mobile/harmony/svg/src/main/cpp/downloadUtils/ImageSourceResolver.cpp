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

#include <fstream>
#include "ImageSourceResolver.h"

namespace rnoh {
namespace svg {

std::string ImageSourceResolver::resolveImageSources(std::shared_ptr<ImageSourceUpdateListener> listener, std::string uri) {
    // 判断是否还在下载中
    if (m_pendingSet.find(uri) != m_pendingSet.end()) {
        removeListener(listener);
        addListenerForURI(uri, listener);
        return "";
    }
    // 生成图像的哈希缓存文件名
    auto hashedFileName = StringUtils::generateHash(uri);

    bool exist = remoteImageSourceMap_.get(hashedFileName);
    if (exist) {
        auto cacheUri = "file://" + fileCacheDir_ + hashedFileName;
        return cacheUri;
    }
    removeListener(listener);
    addListenerForURI(uri, listener);
    return "";
}

void ImageSourceResolver::addListenerForURI(const std::string &uri, std::shared_ptr<ImageSourceUpdateListener> listener) {
    listener->observedUri = uri;
    auto it = m_uriListenersMap.find(uri);
    if (it == m_uriListenersMap.end()) {
        m_uriListenersMap.emplace(uri, std::vector<std::weak_ptr<ImageSourceUpdateListener>>{listener});
        return;
    }
    
    auto alreadyThere = std::find_if(it->second.begin(), it->second.end(),[&listener]  
    (const std::weak_ptr<ImageSourceUpdateListener> &wk )
    {   
        auto sp = wk.lock();
        return sp == listener;

    });
    if (alreadyThere != it->second.end())
    {
        return;
    }
    it->second.push_back(listener);
}

void ImageSourceResolver::removeListenerForURI(const std::string &uri, std::shared_ptr<ImageSourceUpdateListener> listener) {
    auto it = m_uriListenersMap.find(uri);
    if (it == m_uriListenersMap.end()) {
        return;
    }
    auto &listeners = it->second;
    auto listenerPos = std::find_if(listeners.begin(), listeners.end(), [&listener](const std::weak_ptr<ImageSourceUpdateListener> &wk)
    {
        auto sp = wk.lock();
        return sp == listener;

    });
    if (listenerPos != listeners.end()) {
        listeners.erase(listenerPos);
        if (listeners.empty()) {
            m_uriListenersMap.erase(uri);
        }
    }
}

void ImageSourceResolver::removeListener(std::shared_ptr<ImageSourceUpdateListener> listener) {
    if (!listener->observedUri.empty()) {
        removeListenerForURI(listener->observedUri, listener);
    }
}

void ImageSourceResolver::imageDownloadComplete(std::string uri, std::string fileUri) {
    auto pend = m_pendingSet.find(uri);
    if (pend != m_pendingSet.end()) {
        m_pendingSet.erase(uri);
    }
    // 生成图像的哈希缓存文件名
    auto hashedFileName = StringUtils::generateHash(uri);
    DLOG(INFO) << "[SVGImage] remoteImageSourceMap PUT: " << hashedFileName;
    remoteImageSourceMap_.put(hashedFileName, true);
    auto it = m_uriListenersMap.find(uri);
    if (it == m_uriListenersMap.end()) {
        return;
    }

    auto &listeners = it->second;
    for (auto listener : listeners) {
        
        if (auto weaklistener = listener.lock())
        {
            weaklistener->onImageSourceCacheUpdate(fileUri);
            removeListenerForURI(uri, weaklistener);
        }

    }
}

void ImageSourceResolver::imageDownloadFail(std::string uri) {
    if (uri.empty()) {
        return;
    }
    if (!m_pendingSet.empty()) {
        auto pend = m_pendingSet.find(uri);
        if (pend != m_pendingSet.end()) {
            m_pendingSet.erase(uri);
        }
    }
    if (!m_uriListenersMap.empty()) {
        auto it = m_uriListenersMap.find(uri);
        if (it == m_uriListenersMap.end()) {
            return;
        }
        auto &listeners = it->second;
        for (auto listener : listeners) {
            auto weaklistener = listener.lock();
            if (!weaklistener) {
               continue;
            }
			weaklistener->onImageSourceCacheDownloadFileFail();
            removeListenerForURI(uri, weaklistener);
        }
    }
}

} // namespace svg
} // namespace rnoh