import { Pressable, StyleSheet, Text, View, useWindowDimensions } from 'react-native'
import type { BottomTabBarProps } from '@react-navigation/bottom-tabs'

export interface AppTabItem {
  key: string
  label: string
  glyph: string
  badge?: number
}

const TAB_META: Record<string, { glyph: string; label: string }> = {
  Tasks: { glyph: '☰', label: 'Tasks' },
  Sessions: { glyph: '◎', label: 'Sessions' },
  Approvals: { glyph: '✓', label: 'Approvals' },
  Notifications: { glyph: '◔', label: 'Alerts' },
  Settings: { glyph: '⚙', label: 'Settings' },
}

export function AppTabBarView({
  items,
  active,
  onChange,
  bottomInset = 0,
}: {
  items: AppTabItem[]
  active: string
  onChange: (key: string) => void
  bottomInset?: number
}) {
  const { width, height } = useWindowDimensions()
  const vertical = width >= 720 && width > height

  return (
    <View
      style={[
        styles.wrap,
        vertical ? styles.railWrap : styles.bottomWrap,
        !vertical && { paddingBottom: Math.max(bottomInset, 12) },
      ]}
    >
      <View style={[styles.bar, vertical ? styles.rail : styles.pill]}>
        {items.map((item) => {
          const focused = active === item.key
          return (
            <Pressable
              key={item.key}
              accessibilityRole="button"
              accessibilityState={focused ? { selected: true } : {}}
              accessibilityLabel={item.label}
              android_ripple={{ color: '#202433', borderless: true }}
              onPress={() => onChange(item.key)}
              style={[styles.item, vertical ? styles.itemVertical : styles.itemHorizontal, focused && styles.itemActive]}
            >
              <Text style={[styles.glyph, focused && styles.glyphActive]}>{item.glyph}</Text>
              {item.badge ? (
                <View style={styles.badge}>
                  <Text style={styles.badgeText}>{item.badge > 9 ? '9+' : item.badge}</Text>
                </View>
              ) : null}
              <Text style={[styles.label, focused && styles.labelActive]} numberOfLines={1}>
                {item.label}
              </Text>
            </Pressable>
          )
        })}
      </View>
    </View>
  )
}

export function AppTabBar({
  state,
  navigation,
  insets,
  badges,
}: BottomTabBarProps & { badges?: Record<string, number> }) {
  const items = state.routes.map((route) => ({
    key: route.key,
    label: TAB_META[route.name]?.label ?? route.name,
    glyph: TAB_META[route.name]?.glyph ?? '•',
    badge: badges?.[route.name],
  }))
  return (
    <AppTabBarView
      items={items}
      active={state.routes[state.index]?.key ?? ''}
      bottomInset={insets.bottom}
      onChange={(key) => {
        const route = state.routes.find((candidate) => candidate.key === key)
        if (!route) return
        const event = navigation.emit({ type: 'tabPress', target: route.key, canPreventDefault: true })
        if (state.routes[state.index]?.key !== route.key && !event.defaultPrevented) {
          navigation.navigate(route.name)
        }
      }}
    />
  )
}

const styles = StyleSheet.create({
  wrap: { backgroundColor: 'transparent' },
  bottomWrap: { paddingHorizontal: 26, paddingTop: 8 },
  railWrap: { paddingHorizontal: 14, paddingVertical: 16 },
  bar: {
    backgroundColor: 'rgba(15,18,27,0.96)',
    borderWidth: 1,
    borderColor: '#262c3b',
    shadowColor: '#000',
    shadowOpacity: 0.35,
    shadowRadius: 18,
    shadowOffset: { width: 0, height: 8 },
    elevation: 12,
  },
  pill: {
    flexDirection: 'row',
    alignItems: 'center',
    height: 60,
    borderRadius: 30,
    paddingHorizontal: 8,
  },
  rail: {
    width: 84,
    borderRadius: 26,
    paddingVertical: 12,
    gap: 8,
  },
  item: { position: 'relative', alignItems: 'center', justifyContent: 'center', borderRadius: 22 },
  itemHorizontal: { flex: 1, height: 46, gap: 2 },
  itemVertical: { height: 62, gap: 4, marginHorizontal: 6 },
  itemActive: { backgroundColor: '#1d2433' },
  glyph: { color: '#6e7787', fontSize: 16 },
  glyphActive: { color: '#f3f4f6' },
  label: { color: '#6e7787', fontSize: 9.5, letterSpacing: 0.2 },
  labelActive: { color: '#f3f4f6', fontWeight: '600' },
  badge: { position: 'absolute', top: 2, right: 8, minWidth: 15, height: 15, borderRadius: 8, alignItems: 'center', justifyContent: 'center', backgroundColor: '#e05c6b', paddingHorizontal: 3 },
  badgeText: { color: '#ffffff', fontSize: 8.5, fontWeight: '700' },
})
