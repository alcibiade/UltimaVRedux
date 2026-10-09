export interface ShrineStatusEntry {
  index: number
  virtueName: string
  virtueKey: string
  mantra: string
  wordOfPower: string
  worldX: number
  worldY: number
  strBonus: number
  dexBonus: number
  intBonus: number
  karmaBonus: number
  questState: string
  completed: boolean
  active: boolean
  ruined: boolean
}

export interface ShrineStatusData {
  fingerprint: string
  shrines: ShrineStatusEntry[]
  firstActiveIndex: number
  allCompleted: boolean
}
