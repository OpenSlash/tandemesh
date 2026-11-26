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

#include <string>
#include <vector>
#include <folly/dynamic.h>

namespace rnoh {
namespace svg {

namespace DynamicUtils {

/**
 * Convert folly::dynamic to string
 * Handles string, int, double, bool types
 * Returns empty string for other types
 */
inline std::string DynamicToString(const folly::dynamic& d) {
    if (d.isNull()) {
        return "";
    } else if (d.isString()) {
        return d.asString();
    } else if (d.isInt()) {
        return std::to_string(d.asInt());
    } else if (d.isDouble()) {
        return std::to_string(d.asDouble());
    } else if (d.isBool()) {
        return d.asBool() ? "true" : "false";
    } else {
        return "";
    }
}

/**
 * Convert folly::dynamic array to vector of strings
 * Returns empty vector if not an array
 */
inline std::vector<std::string> DynamicToStringVector(const folly::dynamic& d) {
    std::vector<std::string> result;
    if (d.isArray()) {
        for (const auto& item : d) {
            result.push_back(DynamicToString(item));
        }
    }
    return result;
}

} // namespace DynamicUtils
} // namespace svg
} // namespace rnoh
