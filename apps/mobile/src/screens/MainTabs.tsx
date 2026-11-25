import { createBottomTabNavigator } from '@react-navigation/bottom-tabs'
import { TasksScreen } from './TasksScreen'
import { SettingsScreen } from './SettingsScreen'

const Tab = createBottomTabNavigator()

export function MainTabs() {
  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
        tabBarStyle: { backgroundColor: '#0b0d14', borderTopColor: '#202433' },
        tabBarActiveTintColor: '#f3f4f6',
        tabBarInactiveTintColor: '#6e7787',
      }}
    >
      <Tab.Screen name="Tasks" component={TasksScreen} />
      <Tab.Screen name="Settings" component={SettingsScreen} />
    </Tab.Navigator>
  )
}
