import { Avatar as HeroAvatar } from '@heroui/react'
import { useEffect, useState } from 'react'
import { cn, colorFor } from '@/lib/utils'

interface UserAvatarProps {
  name?: string
  src?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const SIZES = { sm: 'size-8 text-[11px]', md: 'size-9 text-xs', lg: 'size-12 text-sm' }

export function UserAvatar({ name, src, size = 'md', className }: UserAvatarProps) {
  const label = (name || '?').trim().slice(0, 1).toUpperCase()
  // Gravatar answers with an image for any address, but an offline or blocked
  // request resolves to a broken <img> that renders as a blank disc; fall back
  // to the initial instead of showing one.
  const [failed, setFailed] = useState(false)
  useEffect(() => setFailed(false), [src])

  return (
    <HeroAvatar className={cn(SIZES[size], 'shrink-0', className)}>
      {src && !failed ? (
        <HeroAvatar.Image src={src} alt="" onError={() => setFailed(true)} />
      ) : null}
      <HeroAvatar.Fallback
        style={{ backgroundColor: colorFor(name || 'anonymous'), color: 'white' }}
        className="font-semibold"
      >
        {label}
      </HeroAvatar.Fallback>
    </HeroAvatar>
  )
}
