# HarmonyOS / RNOH

React Native for OpenHarmony shell for Tandemesh Mobile. Generated from the
RNOH 0.82 template and wired to `TandemeshMobile` (`appKey` in `Index.ets`).

## Prerequisites

- DevEco Studio 26+ with the HarmonyOS SDK
- `ohpm` and `hvigor` from DevEco Studio
- Windows long paths enabled, or build through a short `subst` drive

## Build

Windows cannot handle the deep RNOH HAR paths without long-path support. Map the
mobile directory to a short drive before installing:

```powershell
subst X: D:\project\tandemesh\apps\mobile
cd X:\harmony
```

Install native dependencies:

```powershell
ohpm install --all
```

Generate the RNOH codegen header from the mobile project root:

```powershell
cd X:\
node node_modules/react-native/cli.js codegen-harmony `
  --rnoh-module-path X:\harmony\entry\oh_modules\@rnoh\react-native-openharmony
```

Build the HAP:

```powershell
$env:DEVECO_SDK_HOME = "C:\Program Files\Huawei\DevEco Studio\sdk"
$env:JAVA_HOME = "C:\Program Files\Huawei\DevEco Studio\jbr"
$env:PATH = "$env:JAVA_HOME\bin;" + $env:PATH

hvigorw --mode module -p product=default -p buildMode=debug assembleHap `
  --no-daemon `
  --node-home "C:\Program Files\Huawei\DevEco Studio\tools\node"
```

The unsigned artifact is written to:

```text
harmony/entry/build/default/outputs/default/entry-default-unsigned.hap
```

Signing is not configured in the repository. Use DevEco Studio
`File -> Project Structure -> Signing Configs` to add a signing profile before
installing on a device.

## Bundle

The JS bundle is produced by the mobile project:

```bash
pnpm run dev
```

It writes:

```text
harmony/entry/src/main/resources/rawfile/bundle.harmony.js
```
