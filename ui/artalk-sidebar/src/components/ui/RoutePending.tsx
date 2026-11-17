import { Spinner } from '@heroui/react'

/** Shown while a lazily loaded page chunk is on its way. */
export function RoutePending() {
  return (
    <div className="flex justify-center py-10">
      <Spinner />
    </div>
  )
}
