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

#include <cstdio>
#include <map>
#include <string>
#include <string.h>
#include <stdio.h>
#include <unordered_set>
#include <filesystem>
#include <glog/logging.h>
#include "RemoteCommunicationKit/rcp.h"
#include "ImageSourceResolver.h"
#include "utils/StringUtils.h"

namespace fs = std::filesystem;
namespace rnoh {
namespace svg {
class HttpTaskProcessor {
public:
    HttpTaskProcessor() { taskList_.push_back(this); }

    ~HttpTaskProcessor() {
        if (request_ != nullptr) {
            // 释放请求资源
            HMS_Rcp_DestroyRequest(request_);
            request_ = nullptr;
        }
    }

    static void clearTaskList() { taskList_.clear(); }

    void createParentPath(const fs::path &filePath);

    size_t saveImage(const char *buffer, uint32_t length);

    void markStart();

    void markEnd();

    bool keep() { return !finished_; }

    void echoSpendTime() {
        DLOG(INFO) << "[SVGImage] HttpTaskProcessor Spend Time(ns): " << endTime_.count() - startTime_.count();
    }

    bool initializeRequestAndSession(const char *uri, const std::map<std::string, std::string> *headers = nullptr);
    void setResponseCallback();

    void launchHttpRequest(const char *uri);
    void launchHttpRequestWithHeaders(const char *uri, const std::map<std::string, std::string> &headers);


    void pushThreadID(std::string id);

    static void waitAllTask();
    static void destroyAllTask();
    static void echoThreadIds();

    std::shared_ptr<ImageSourceResolver> instance_;

    std::string uri_;
    std::string filePath_;

private:
    static std::vector<HttpTaskProcessor *> taskList_;
    static std::unordered_set<std::string> threadIds_;

    FILE *fp_ = nullptr;

    std::chrono::nanoseconds startTime_;
    std::chrono::nanoseconds endTime_;

    bool finished_ = false;

    Rcp_Request *request_ = nullptr;
    Rcp_Session *session_ = nullptr;

    Rcp_ResponseCallbackObject responseCallback_;
};

} // namespace svg
} // namespace rnoh
