import type { ComponentType } from 'react'
import { useNavigation, useRoute } from '@react-navigation/native'
import { RouterContext, type RouteName } from '../lib/RouterContext'

export function withNativeRouter(Component: ComponentType) {
  return function NativeRoutedScreen() {
    const navigation = useNavigation<any>()
    const route = useRoute<any>()
    return (
      <RouterContext.Provider
        value={{
          params: route.params,
          navigate: (name: RouteName, params?: Record<string, string>) => navigation.navigate(name, params),
          back: () => navigation.goBack(),
        }}
      >
        <Component />
      </RouterContext.Provider>
    )
  }
}
