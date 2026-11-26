#pragma once

// This file was generated.

#include "RNOHCorePackage/ComponentBinders/ViewComponentJSIBinder.h"

namespace rnoh {
class RNSVGFeOffsetJSIBinder : public ViewComponentJSIBinder {
  protected:
    facebook::jsi::Object createNativeProps(facebook::jsi::Runtime &rt) override {
        auto object = ViewComponentJSIBinder::createNativeProps(rt);
        object.setProperty(rt, "name", true);
        object.setProperty(rt, "x", true);
        object.setProperty(rt, "y", true);
        object.setProperty(rt, "width", true);
        object.setProperty(rt, "height", true);
        object.setProperty(rt, "result", true);
        object.setProperty(rt, "in1", true);
        object.setProperty(rt, "dx", true);
        object.setProperty(rt, "dy", true);
        object.setProperty(rt, "pointerEvents", true);

        return object;
    }
};
} // namespace rnoh
