import {mergeConfig} from '@/utils'

export const DEFAULT_CONFIG = {
  minGiftPrice: 0, // ￥0
  exchangeRate: 1, // 1 -> ￥1
  showDanmaku: true,
  showFollow: true,
  showJoin: true,
  showQuit: false,
  showGift: true,
  showGiftPrice: true,
  showACCoinInstead: false,
  showLove: true,
  showGiftName: false,
  mergeSimilarDanmaku: true,
  mergeSimilarOther: true,
  mergeGift: true,
  maxNumber: 60,

  blockGiftDanmaku: false,
  blockLevel: 0,
  blockNewbie: false,
  blockNotMobileVerified: false,
  blockKeywords: '',
  blockUsers: '',
  blockMedalLevel: 0,

  autoTranslate: false
}

export function setLocalConfig (config) {
  config = mergeConfig(config, DEFAULT_CONFIG)
  window.localStorage.config = JSON.stringify(config)
}

export function getLocalConfig () {
  if (!window.localStorage.config) {
    return DEFAULT_CONFIG
  }
  return mergeConfig(JSON.parse(window.localStorage.config), DEFAULT_CONFIG)
}
