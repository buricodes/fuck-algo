import { ACTIVITY_CAP, REPEAT_WINDOW, STATS_DAYS, TOPIC_DAYS } from '@/shared/defaults'
import { update } from '@/shared/storage'
import type { ActivityEntry } from '@/shared/types'
import { dayKey, newId } from '@/shared/util'

let chain: Promise<unknown> = Promise.resolve()
function serial<T>(task: () => Promise<T>): Promise<T> {
  const run = chain.then(task, task)
  chain = run.catch(() => undefined)
  return run
}

function keepRecentDays<T>(days: Record<string, T>, keep: number): Record<string, T> {
  const cutoff = dayKey(new Date(Date.now() - keep * 86_400_000))
  return Object.fromEntries(Object.entries(days).filter(([day]) => day > cutoff))
}

export function recordBlock(entry: Omit<ActivityEntry, 'id' | 'at' | 'undone'>) {
  return serial(async () => {
    const at = new Date().toISOString()
    let duplicate = false
    await update('activity', (log) => {
      duplicate = log.some((e) => e.key === entry.key && e.at.slice(0, 10) === at.slice(0, 10))
      if (duplicate) return log
      return [{ ...entry, id: newId(), at, undone: false }, ...log].slice(0, ACTIVITY_CAP)
    })
    if (duplicate) return
    await update('stats', (stats) => {
      const today = dayKey()
      const day = stats.days[today] ?? { spoiler: 0, repeat: 0, keyword: 0 }
      return {
        days: keepRecentDays(
          { ...stats.days, [today]: { ...day, [entry.kind]: day[entry.kind] + 1 } },
          STATS_DAYS,
        ),
      }
    })
  })
}

export function recordSeen(key: string, topic: string) {
  return serial(() =>
    update('feed', (feed) => {
      if (feed.recent.some((seen) => seen.key === key)) return feed
      const today = dayKey()
      const day = { ...feed.topicsByDay[today] }
      day[topic] = (day[topic] ?? 0) + 1
      return {
        recent: [...feed.recent, { key, topic, at: new Date().toISOString() }].slice(
          -REPEAT_WINDOW,
        ),
        topicsByDay: keepRecentDays({ ...feed.topicsByDay, [today]: day }, TOPIC_DAYS),
      }
    }),
  )
}
