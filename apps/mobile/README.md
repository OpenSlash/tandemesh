# Tandemesh Mobile

React Native 0.82 shell for Android and iOS. HarmonyOS uses RNOH 0.82 and is added in a later step.

## Pairing

The app pairs with a Tandemesh plane using a one-time code generated in
`Settings -> Mobile devices` on the desktop or web UI.

```text
tandemesh://pair?host=http://192.168.1.10:8080&code=ABCD-EFGH
```

## Run

```bash
pnpm install
pnpm start
pnpm android
```

The current scaffold keeps the bearer token in memory. Replace `src/lib/session.ts`
with Keychain / Keystore / HarmonyOS secure storage before shipping.
