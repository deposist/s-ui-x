import {
  appMenu,
  appMenuGroups,
  singBoxSettingsPaths,
  type MenuCountKey,
  type MenuGroup,
  type MenuItem,
} from '@/layouts/menu'

export type NexusCountKey = MenuCountKey
export type NexusMenuItem = MenuItem
export type NexusMenuGroup = MenuGroup

export const nexusMenuGroups = appMenuGroups
export const nexusMenu = appMenu
export const nexusSingBoxSettingsPaths = singBoxSettingsPaths
