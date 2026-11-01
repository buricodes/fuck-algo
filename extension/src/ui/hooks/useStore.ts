import { useCallback, useEffect, useState } from 'react'

import { read, update, watch } from '@/shared/storage'
import type { Store } from '@/shared/storage'

export function useStore<K extends keyof Store>(key: K) {
  const [value, setValue] = useState<Store[K] | null>(null)

  useEffect(() => {
    let alive = true
    void read(key).then((v) => alive && setValue(v))
    const stop = watch(key, (v) => setValue(v))
    return () => {
      alive = false
      stop()
    }
  }, [key])

  const change = useCallback(
    (fn: (current: Store[K]) => Store[K]) => {
      void update(key, fn).then(setValue)
    },
    [key],
  )

  return [value, change] as const
}
