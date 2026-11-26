import { createContext, useCallback, useContext, useMemo, useState, type PropsWithChildren } from 'react'
import { NativeModules } from 'react-native'

export type Lang = 'en' | 'zh'

const en = {
  'tabs.tasks': 'Tasks',
  'tabs.sessions': 'Sessions',
  'tabs.approvals': 'Approvals',
  'tabs.alerts': 'Alerts',
  'tabs.settings': 'Settings',
  'common.back': 'Back',
  'common.cancel': 'Cancel',
  'common.create': 'Create',
  'common.creating': 'Creating…',
  'common.send': 'Send',
  'common.sending': 'Sending…',
  'common.thinking': 'Thinking…',
  'common.responding': 'Responding…',
  'common.loading': 'Loading…',
  'tasks.empty': 'No tasks in this workspace.',
  'tasks.filter.all': 'All',
  'tasks.filter.open': 'Open',
  'tasks.filter.progress': 'In progress',
  'tasks.filter.done': 'Done',
  'sessions.title': 'Sessions',
  'sessions.new': 'New session',
  'sessions.newTitle': 'New session',
  'sessions.titlePlaceholder': 'Session title',
  'sessions.empty': 'No agent sessions yet.',
  'approvals.title': 'Approvals',
  'approvals.empty': 'No pending approvals.',
  'notifications.title': 'Notifications',
  'notifications.empty': 'Nothing new.',
  'settings.title': 'Settings',
  'settings.plane': 'Connected plane',
  'settings.workspace': 'Workspace',
  'settings.workspaces': 'Workspaces',
  'settings.version': 'App version',
  'settings.platform': 'Platform',
  'settings.language': 'Language',
  'settings.theme': 'Theme',
  'settings.theme.system': 'System',
  'settings.theme.light': 'Light',
  'settings.theme.dark': 'Dark',
  'settings.unpair': 'Unpair this device',
  'settings.unpairTitle': 'Unpair device?',
  'settings.unpairBody': 'The bearer token will be deleted from this device.',
  'settings.unpairAction': 'Unpair',
  'session.messagePlaceholder': 'Message the agent…',
  'session.showThinking': 'Show thinking',
  'session.hideThinking': 'Hide thinking',
  'session.empty': 'No messages yet.',
  'pair.title': 'Pair this device',
  'pair.hint': 'On the desktop open Settings → Mobile devices, generate a code, then enter it below.',
  'pair.host': 'Host',
  'pair.code': 'Pairing code',
  'pair.device': 'Device name',
  'pair.action': 'Pair',
  'pair.pairing': 'Pairing…',
}

const zh: typeof en = {
  'tabs.tasks': '任务',
  'tabs.sessions': '会话',
  'tabs.approvals': '审批',
  'tabs.alerts': '通知',
  'tabs.settings': '设置',
  'common.back': '返回',
  'common.cancel': '取消',
  'common.create': '创建',
  'common.creating': '创建中…',
  'common.send': '发送',
  'common.sending': '发送中…',
  'common.thinking': '思考中…',
  'common.responding': '回复中…',
  'common.loading': '加载中…',
  'tasks.empty': '当前工作区没有任务。',
  'tasks.filter.all': '全部',
  'tasks.filter.open': '待处理',
  'tasks.filter.progress': '进行中',
  'tasks.filter.done': '已完成',
  'sessions.title': '会话',
  'sessions.new': '新建会话',
  'sessions.newTitle': '新建会话',
  'sessions.titlePlaceholder': '会话标题',
  'sessions.empty': '还没有 Agent 会话。',
  'approvals.title': '审批',
  'approvals.empty': '没有待审批项。',
  'notifications.title': '通知',
  'notifications.empty': '暂无新通知。',
  'settings.title': '设置',
  'settings.plane': '已连接控制面',
  'settings.workspace': '工作区',
  'settings.workspaces': '所有工作区',
  'settings.version': '应用版本',
  'settings.platform': '平台',
  'settings.language': '语言',
  'settings.theme': '主题',
  'settings.theme.system': '跟随系统',
  'settings.theme.light': '浅色',
  'settings.theme.dark': '深色',
  'settings.unpair': '解除本机配对',
  'settings.unpairTitle': '解除配对？',
  'settings.unpairBody': '本机保存的访问令牌将被删除。',
  'settings.unpairAction': '解除',
  'session.messagePlaceholder': '给 Agent 发消息…',
  'session.showThinking': '查看思考',
  'session.hideThinking': '收起思考',
  'session.empty': '还没有消息。',
  'pair.title': '配对本机',
  'pair.hint': '在桌面端打开 设置 → 移动设备，生成配对码后填入下方。',
  'pair.host': '主机',
  'pair.code': '配对码',
  'pair.device': '设备名称',
  'pair.action': '配对',
  'pair.pairing': '配对中…',
}

const dictionaries: Record<Lang, typeof en> = { en, zh }

export type TranslationKey = keyof typeof en

function detectLang(): Lang {
  try {
    const locale =
      NativeModules.I18nManager?.localeIdentifier ??
      NativeModules.SettingsManager?.settings?.AppleLocale ??
      Intl.DateTimeFormat().resolvedOptions().locale
    return String(locale).toLowerCase().startsWith('zh') ? 'zh' : 'en'
  } catch {
    return 'en'
  }
}

interface I18nValue {
  lang: Lang
  setLang: (lang: Lang) => void
  t: (key: TranslationKey, params?: Record<string, string | number>) => string
}

const I18nContext = createContext<I18nValue | null>(null)

export function I18nProvider({ children }: PropsWithChildren) {
  const [lang, setLang] = useState<Lang>(detectLang)
  const t = useCallback(
    (key: TranslationKey, params?: Record<string, string | number>) => {
      let text: string = dictionaries[lang][key] ?? dictionaries.en[key] ?? key
      if (params) {
        for (const [name, value] of Object.entries(params)) {
          text = text.replace(`{${name}}`, String(value))
        }
      }
      return text
    },
    [lang],
  )
  const value = useMemo(() => ({ lang, setLang, t }), [lang, t])
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n(): I18nValue {
  const value = useContext(I18nContext)
  if (!value) throw new Error('I18nProvider is missing')
  return value
}
