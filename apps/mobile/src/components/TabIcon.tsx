import type { ComponentType } from 'react'
import { Platform, StyleSheet, View } from 'react-native'

export type TabIconName = 'tasks' | 'sessions' | 'approvals' | 'alerts' | 'settings'

export function TabIcon({ name, active }: { name: TabIconName; active: boolean }) {
  const color = active ? '#f3f4f6' : '#6e7787'
  if (name === 'tasks') {
    return (
      <View style={styles.tasks}>
        <View style={[styles.taskBar, { backgroundColor: color, width: 18 }]} />
        <View style={[styles.taskBar, { backgroundColor: color, width: 13 }]} />
        <View style={[styles.taskBar, { backgroundColor: color, width: 16 }]} />
      </View>
    )
  }
  if (name === 'sessions') {
    return (
      <View style={styles.session}>
        <View style={[styles.sessionBubble, { borderColor: color }]}>
          <View style={[styles.sessionDot, { backgroundColor: color }]} />
          <View style={[styles.sessionDot, { backgroundColor: color }]} />
          <View style={[styles.sessionDot, { backgroundColor: color }]} />
        </View>
        <View style={[styles.sessionTail, { backgroundColor: color }]} />
      </View>
    )
  }
  if (name === 'approvals') {
    return (
      <View style={styles.approval}>
        <View style={[styles.checkShort, { backgroundColor: color }]} />
        <View style={[styles.checkLong, { backgroundColor: color }]} />
      </View>
    )
  }
  if (name === 'alerts') {
    return (
      <View style={styles.alert}>
        <View style={[styles.alertBell, { borderColor: color }]} />
        <View style={[styles.alertClapper, { backgroundColor: color }]} />
      </View>
    )
  }
  return (
    <View style={styles.settings}>
      <View style={[styles.settingsRing, { borderColor: color }]} />
      <View style={[styles.settingsCore, { backgroundColor: color }]} />
      <View style={[styles.spoke, styles.spokeTop, { backgroundColor: color }]} />
      <View style={[styles.spoke, styles.spokeBottom, { backgroundColor: color }]} />
      <View style={[styles.spoke, styles.spokeLeft, { backgroundColor: color }]} />
      <View style={[styles.spoke, styles.spokeRight, { backgroundColor: color }]} />
    </View>
  )
}

const styles = StyleSheet.create({
  tasks: { width: 18, height: 16, justifyContent: 'center', gap: 3 },
  taskBar: { height: 2, borderRadius: 1 },
  session: { width: 18, height: 16, alignItems: 'center', justifyContent: 'center' },
  sessionBubble: {
    width: 16,
    height: 12,
    borderRadius: 4,
    borderWidth: 1.5,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 2,
  },
  sessionDot: { width: 2, height: 2, borderRadius: 1 },
  sessionTail: {
    position: 'absolute',
    left: 3,
    bottom: 0,
    width: 4,
    height: 4,
    transform: [{ rotate: '45deg' }],
    borderRadius: 1,
  },
  approval: { width: 18, height: 16, alignItems: 'center', justifyContent: 'center' },
  checkShort: {
    position: 'absolute',
    width: 7,
    height: 2,
    borderRadius: 1,
    transform: [{ rotate: '45deg' }, { translateX: -3 }, { translateY: 2 }],
  },
  checkLong: {
    position: 'absolute',
    width: 13,
    height: 2,
    borderRadius: 1,
    transform: [{ rotate: '-45deg' }, { translateX: 3 }, { translateY: 1 }],
  },
  alert: { width: 18, height: 16, alignItems: 'center', justifyContent: 'center' },
  alertBell: { width: 13, height: 13, borderRadius: 7, borderWidth: 1.5 },
  alertClapper: {
    position: 'absolute',
    bottom: 0,
    width: 4,
    height: 2,
    borderRadius: 1,
  },
  settings: { width: 18, height: 16, alignItems: 'center', justifyContent: 'center' },
  settingsRing: { width: 12, height: 12, borderRadius: 6, borderWidth: 1.5 },
  settingsCore: { position: 'absolute', width: 4, height: 4, borderRadius: 2 },
  spoke: { position: 'absolute', borderRadius: 1 },
  spokeTop: { width: 2, height: 4, top: 0 },
  spokeBottom: { width: 2, height: 4, bottom: 0 },
  spokeLeft: { width: 4, height: 2, left: 0 },
  spokeRight: { width: 4, height: 2, right: 0 },
})
