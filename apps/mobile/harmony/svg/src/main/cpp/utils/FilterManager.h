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
#include <unordered_map>
#include <memory>
#include <glog/logging.h>

namespace rnoh {
namespace svg {

// FeColorMatrix信息结构
struct FeColorMatrixInfo {
    std::string type = "saturate";
    std::vector<float> values = {1.0f};
    std::string in = "SourceGraphic";  // 默认输入源为 SourceGraphic
    
    FeColorMatrixInfo() = default;
    FeColorMatrixInfo(const std::string& t, const std::vector<float>& v) : type(t), values(v) {}
    FeColorMatrixInfo(const std::string& t, const std::vector<float>& v, const std::string& inputSource) 
        : type(t), values(v), in(inputSource) {}
    
    // 获取饱和度值（当type为"saturate"时）
    float getSaturationValue() const {
        if (type == "saturate" && !values.empty()) {
            return values[0];
        }
        return 1.0f; // 默认饱和度
    }
    
    // 获取颜色矩阵（当type为"matrix"时）
    std::vector<float> getColorMatrix() const {
        if (type == "matrix" && values.size() >= 20) {
            return values;
        }
        // 返回单位矩阵
        return {1,0,0,0,0, 0,1,0,0,0, 0,0,1,0,0, 0,0,0,1,0};
    }
    
    // 检查是否为有效的FeColorMatrix
    bool isValid() const {
        if (type == "saturate") {
            return !values.empty();
        } else if (type == "matrix") {
            return values.size() >= 20;
        } else if (type == "hueRotate") {
            return !values.empty();
        } else if (type == "luminanceToAlpha") {
            return true; // 不需要values
        }
        return false;
    }
    
    // 获取输入源类型
    std::string getInputSource() const {
        return in;
    }
};

// FeBlend信息结构
struct FeBlendInfo {
    std::string in = "SourceGraphic";   // 第一个输入源
    std::string in2 = "";               // 第二个输入源
    std::string mode = "normal";        // 混合模式
    
    FeBlendInfo() = default;
    FeBlendInfo(const std::string& input1, const std::string& input2, const std::string& blendMode) 
        : in(input1), in2(input2), mode(blendMode) {}
    
    // 检查是否为有效的混合模式
    bool isValidMode() const {
        return mode == "normal" || mode == "multiply" || mode == "screen" || 
               mode == "darken" || mode == "lighten";
    }
};

// Filter管理器类
class FilterManager {
public:
    // 单例模式
    static FilterManager& getInstance() {
        static FilterManager instance;
        return instance;
    }
    
    // 注册Filter和FeColorMatrix的映射
    void registerFilter(const std::string& filterId, std::shared_ptr<FeColorMatrixInfo> feColorMatrixInfo) {
        if (!filterId.empty() && feColorMatrixInfo) {
            filterMap_[filterId] = feColorMatrixInfo;
        }
    }
    
    // 根据Filter ID获取FeColorMatrix信息
    std::shared_ptr<FeColorMatrixInfo> getFeColorMatrixInfo(const std::string& filterId) const {
        auto it = filterMap_.find(filterId);
        if (it != filterMap_.end()) {
            return it->second;
        }
        return nullptr;
    }
    
    // 检查Filter是否存在
    bool hasFilter(const std::string& filterId) const {
        return filterMap_.find(filterId) != filterMap_.end();
    }
    
    // 清除所有Filter映射
    void clearAllFilters() {
        filterMap_.clear();
        blendFilterMap_.clear();
    }
    
    // 移除指定Filter
    void removeFilter(const std::string& filterId) {
        filterMap_.erase(filterId);
        blendFilterMap_.erase(filterId);
    }
    
    // ========== FeBlend 相关方法 ==========
    
    // 注册FeBlend过滤器
    void registerBlendFilter(const std::string& filterId, std::shared_ptr<FeBlendInfo> feBlendInfo) {
        if (!filterId.empty() && feBlendInfo) {
            blendFilterMap_[filterId] = feBlendInfo;
        }
    }
    
    // 获取FeBlend信息
    std::shared_ptr<FeBlendInfo> getFeBlendInfo(const std::string& filterId) const {
        auto it = blendFilterMap_.find(filterId);
        if (it != blendFilterMap_.end()) {
            return it->second;
        }
        return nullptr;
    }
    
    // 检查FeBlend Filter是否存在
    bool hasBlendFilter(const std::string& filterId) const {
        return blendFilterMap_.find(filterId) != blendFilterMap_.end();
    }
    
    // 获取所有Filter ID
    std::vector<std::string> getAllFilterIds() const {
        std::vector<std::string> ids;
        for (const auto& pair : filterMap_) {
            ids.push_back(pair.first);
        }
        return ids;
    }
    
    // 解析Filter URL（例如："url(#myFilter)" -> "myFilter"）
    static std::string parseFilterUrl(const std::string& filterUrl) {
        if (filterUrl.empty()) return "";
        
        // 查找"url(#"
        size_t start = filterUrl.find("url(#");
        if (start != std::string::npos) {
            start += 5; // 跳过"url(#"
            size_t end = filterUrl.find(")", start);
            if (end != std::string::npos) {
                return filterUrl.substr(start, end - start);
            }
        }
        
        // 如果没有找到"url(#"格式，检查是否直接是ID
        if (filterUrl.front() == '#') {
            return filterUrl.substr(1);
        }
        
        return filterUrl;
    }

private:
    FilterManager() = default;
    ~FilterManager() = default;
    FilterManager(const FilterManager&) = delete;
    FilterManager& operator=(const FilterManager&) = delete;
    
    std::unordered_map<std::string, std::shared_ptr<FeColorMatrixInfo>> filterMap_;
    std::unordered_map<std::string, std::shared_ptr<FeBlendInfo>> blendFilterMap_;
};

} // namespace svg
} // namespace rnoh
