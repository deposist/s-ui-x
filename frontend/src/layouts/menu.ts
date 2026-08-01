export type MenuCountKey =
  | 'inbounds'
  | 'clients'
  | 'outbounds'
  | 'endpoints'
  | 'services'
  | 'tlsConfigs'

export interface MenuItem {
  title: string
  path: string
  classicIcon: string
  nexusIcon: string
  singBoxSettings?: boolean
  countKey?: MenuCountKey
}

export interface MenuGroup {
  labelKey?: string
  items: MenuItem[]
}

export const appMenuGroups: MenuGroup[] = [
  {
    items: [
      { title: 'pages.home', path: '/', classicIcon: 'mdi-home', nexusIcon: 'lucide:layout-grid' },
    ],
  },
  {
    labelKey: 'nav.groups.proxy',
    items: [
      { title: 'pages.inbounds', path: '/inbounds', classicIcon: 'mdi-cloud-download', nexusIcon: 'lucide:zap', singBoxSettings: true, countKey: 'inbounds' },
      { title: 'pages.clients', path: '/clients', classicIcon: 'mdi-account-multiple', nexusIcon: 'lucide:users', countKey: 'clients' },
      { title: 'pages.outbounds', path: '/outbounds', classicIcon: 'mdi-cloud-upload', nexusIcon: 'lucide:arrow-up-right', singBoxSettings: true, countKey: 'outbounds' },
      { title: 'pages.endpoints', path: '/endpoints', classicIcon: 'mdi-cloud-tags', nexusIcon: 'lucide:globe', singBoxSettings: true, countKey: 'endpoints' },
      { title: 'pages.services', path: '/services', classicIcon: 'mdi-server', nexusIcon: 'lucide:server', singBoxSettings: true, countKey: 'services' },
    ],
  },
  {
    labelKey: 'nav.groups.network',
    items: [
      { title: 'pages.tls', path: '/tls', classicIcon: 'mdi-certificate', nexusIcon: 'lucide:lock', singBoxSettings: true, countKey: 'tlsConfigs' },
      { title: 'pages.rules', path: '/rules', classicIcon: 'mdi-routes', nexusIcon: 'lucide:list', singBoxSettings: true },
      { title: 'pages.dns', path: '/dns', classicIcon: 'mdi-dns', nexusIcon: 'lucide:network', singBoxSettings: true },
    ],
  },
  {
    labelKey: 'nav.groups.integrations',
    items: [
      { title: 'pages.telegram', path: '/telegram', classicIcon: 'mdi-send', nexusIcon: 'lucide:send' },
      { title: 'pages.paidSub', path: '/paid-subscriptions', classicIcon: 'mdi-cash-multiple', nexusIcon: 'lucide:credit-card' },
    ],
  },
  {
    labelKey: 'nav.groups.system',
    items: [
      { title: 'pages.admins', path: '/admins', classicIcon: 'mdi-account-tie', nexusIcon: 'lucide:user-cog' },
      { title: 'pages.audit', path: '/audit', classicIcon: 'mdi-shield-search', nexusIcon: 'lucide:file-text' },
      { title: 'pages.settings', path: '/settings', classicIcon: 'mdi-cog', nexusIcon: 'lucide:settings' },
    ],
  },
  {
    labelKey: 'nav.groups.support',
    items: [
      { title: 'pages.donations', path: '/donations', classicIcon: 'mdi-heart', nexusIcon: 'lucide:heart' },
    ],
  },
]

export const appMenu = appMenuGroups.flatMap(group => group.items)

export const singBoxSettingsPaths = appMenu
  .filter(item => item.singBoxSettings)
  .map(item => item.path)
