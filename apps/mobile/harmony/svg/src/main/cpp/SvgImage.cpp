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

#include "SvgImage.h"
#include "SvgFilter.h"
#include "utils/StringUtils.h"
#include "properties/ViewBox.h"
#include <multimedia/image_framework/image_mdk_common.h>
#include <multimedia/image_framework/image/image_source_native.h>
#include <multimedia/image_framework/image_pixel_map_mdk.h>
#include <glog/logging.h>
#include <native_drawing/drawing_image.h>
#include <native_drawing/drawing_rect.h>
#include <native_drawing/drawing_pixel_map.h>
#include <native_drawing/drawing_sampling_options.h>
#include <native_drawing/drawing_filter.h>
#include <rawfile/raw_file.h>

const char *ASSET_PREFIX = "asset://";
const char *RAWFILE_PREFIX = "assets/";

namespace rnoh {
namespace svg {

void SvgImage::OnDraw(OH_Drawing_Canvas *canvas) {
    auto uriString = imageAttribute_.src.uri;

    if (uriString.empty()) {
        DLOG(WARNING) << "[SvgImage] imageAttribute_.src.uri is empty!";
        return;
    }

    double width =
        imageAttribute_.width.Value() == 0 ? imageAttribute_.src.size.width : relativeOnWidth(imageAttribute_.width);
    double height = imageAttribute_.height.Value() == 0 ? imageAttribute_.src.size.height
                                                        : relativeOnHeight(imageAttribute_.height);
    double x = relativeOnWidth(imageAttribute_.x);
    double y = relativeOnHeight(imageAttribute_.y);

    if (!uriString.empty() && uriString.find("http", 0) != 0) {
        OH_ImageSourceNative *res;
        OH_PixelmapNative *pixelMap;
        OH_DecodingOptions *options;

        char *srcUri = const_cast<char *>(uriString.c_str());

        OH_DecodingOptions_Create(&options);

        Image_ErrorCode createFromUriStatus;
        if (uriString.find(ASSET_PREFIX, 0) == 0) {
            auto file = uriString.replace(0, 8, RAWFILE_PREFIX);
            DLOG(INFO) << "[SvgImage] file: " << file;
            auto rawFile = OH_ResourceManager_OpenRawFile(mgr_, file.c_str());
            if (rawFile == nullptr) {
                LOG(ERROR) << "[SvgImage] get rawfile fail";
                return;
            }
            RawFileDescriptor descriptor;
            auto getRawFileStatus = OH_ResourceManager_GetRawFileDescriptor(rawFile, descriptor);
            if (!getRawFileStatus) {
                LOG(ERROR) << "[SvgImage] get rawfile descriptor fail";
                return;
            }
            createFromUriStatus = OH_ImageSourceNative_CreateFromRawFile(&descriptor, &res);
        } else {
            createFromUriStatus = OH_ImageSourceNative_CreateFromUri(srcUri, uriString.size(), &res);
        }
        auto createPixelmapStatus = OH_ImageSourceNative_CreatePixelmap(res, options, &pixelMap);
        DLOG(INFO) << "[SvgImage] code: " << createPixelmapStatus;

        if (createPixelmapStatus == IMAGE_SUCCESS) {
            /* Temporarily disable the feature to set opacity.
             * Currently, there is an issue where setting opacity on
             * transparent pixels results in them turning black.
             */
            if (LessNotEqual(attributes_.opacity, 1.0f)) {
                OH_PixelmapNative_Opacity(pixelMap, attributes_.opacity);
            }

            // get the real width and height from pixelmap(OH_PixelmapNative *).
            OH_Pixelmap_ImageInfo *info;
            OH_PixelmapImageInfo_Create(&info);
            OH_PixelmapNative_GetImageInfo(pixelMap, info);

            // get the resource properties: width and height
            uint32_t *imageWidth_ = new uint32_t;
            uint32_t *imageHeight_ = new uint32_t;
            OH_PixelmapImageInfo_GetWidth(info, imageWidth_);
            OH_PixelmapImageInfo_GetHeight(info, imageHeight_);

            auto imageWidth = static_cast<double>(*imageWidth_);
            auto imageHeight = static_cast<double>(*imageHeight_);
            delete imageWidth_;
            delete imageHeight_;

            const auto vbRect = Rect(0, 0, imageWidth, imageHeight);
            const auto renderRect = Rect(x, y, width, height);

            drawing::Matrix transformMatrix;
            if (vbRect.IsValid()) {
                transformMatrix =
                    ViewBox::getTransform(vbRect, renderRect, imageAttribute_.align, imageAttribute_.meetOrSlice);
            }

            // set source Rect and destination Rect
            OH_Drawing_PixelMap *ohPixelMap = OH_Drawing_PixelMapGetFromOhPixelMapNative(pixelMap);
            drawing::Rect srcPixelMap(0, 0, imageWidth, imageHeight);
            auto dstPixelMap = transformMatrix.MapRect(srcPixelMap);

            // set SamplingOptions
            OH_Drawing_FilterMode filterMode = FILTER_MODE_NEAREST;
            OH_Drawing_MipmapMode mipMapMode = MIPMAP_MODE_NONE;
            OH_Drawing_SamplingOptions *samplingOptions = OH_Drawing_SamplingOptionsCreate(filterMode, mipMapMode);

            // Draw picture by type OH_Drawing_PixelMap
            OH_Drawing_CanvasSave(canvas);
            
            drawing::Rect clipRect(x, y, x + width, y + height);
            OH_Drawing_CanvasClipRect(canvas, clipRect.get(), OH_Drawing_CanvasClipOp::INTERSECT, true);
            
            // Apply SVG filter if specified
            if (!attributes_.filterId.empty()) {
                DLOG(INFO) << "[SvgImage] Applying filter: " << attributes_.filterId;
                auto svgContext = GetContext();
                if (svgContext) {
                    auto filterNode = svgContext->GetSvgNodeById(attributes_.filterId);
                    auto svgFilter = std::dynamic_pointer_cast<SvgFilter>(filterNode);
                    if (svgFilter) {
                        // Create a temporary brush and apply filter to it
                        drawing::Brush tmpBrush;
                        svgFilter->ApplyToBrush(tmpBrush, "image");
                        
                        // Attach the filtered brush to canvas
                        OH_Drawing_CanvasAttachBrush(canvas, tmpBrush.get());
                        DLOG(INFO) << "[SvgImage] Filter applied to brush successfully";
                    }
                }
            }
            
            // Draw the pixelmap
            OH_Drawing_CanvasDrawPixelMapRect(canvas, ohPixelMap, srcPixelMap.get(), dstPixelMap.get(),
                                              samplingOptions);
            
            // Detach brush if filter was applied
            if (!attributes_.filterId.empty()) {
                OH_Drawing_CanvasDetachBrush(canvas);
            }
            
            OH_Drawing_CanvasRestore(canvas);

            // clear data
            OH_PixelmapImageInfo_Release(info);
            OH_Drawing_SamplingOptionsDestroy(samplingOptions);
            OH_Drawing_PixelMapDissolve(ohPixelMap);
            OH_PixelmapNative_Release(pixelMap);
        }
        if (createFromUriStatus == IMAGE_SUCCESS) {
            OH_ImageSourceNative_Release(res);
        }
        OH_DecodingOptions_Release(options);
    }
}

Rect SvgImage::AsBounds() {
    double width =
        imageAttribute_.width.Value() == 0 ? imageAttribute_.src.size.width : relativeOnWidth(imageAttribute_.width);
    double height = imageAttribute_.height.Value() == 0 ? imageAttribute_.src.size.height
                                                        : relativeOnHeight(imageAttribute_.height);
    double x = relativeOnWidth(imageAttribute_.x);
    double y = relativeOnHeight(imageAttribute_.y);
    return Rect(x, y, width, height);
}

} // namespace svg
} // namespace rnoh
