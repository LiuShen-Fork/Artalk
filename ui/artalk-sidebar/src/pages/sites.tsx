import { Button, Skeleton } from '@heroui/react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Globe2, Plus } from 'lucide-react'
import { useState } from 'react'
import { EmptyState } from '@/components/ui/EmptyState'
import { PageHeader } from '@/components/ui/PageHeader'
import { SiteCreateDialog, SiteEditorDialog } from '@/components/SiteDialogs'
import { t } from '@/lib/i18n'
import type { SiteData } from '@/lib/api'
import { colorFor, formatNumber } from '@/lib/utils'
import { getArtalk } from '@/lib/artalk'
import { session } from '@/lib/session'

export function SitesPage() {
  const queryClient = useQueryClient()
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<SiteData>()

  const { data, isPending } = useQuery({
    queryKey: ['sites'],
    queryFn: async () => {
      const res = await getArtalk().ctx.getApi().sites.getSites()
      return (res.data?.sites ?? []) as SiteData[]
    },
  })

  const sites = data ?? []
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['sites'] })
    session.refreshSites()
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title={t('siteManage')}
        description={t('siteCount', { count: sites.length })}
        actions={
          <Button size="sm" variant="primary" onPress={() => setCreating(true)}>
            <Plus className="size-4" aria-hidden />
            {t('createSite')}
          </Button>
        }
      />

      {isPending ? (
        <div className="grid grid-cols-2 gap-2.5">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-[92px] rounded-xl" />
          ))}
        </div>
      ) : sites.length ? (
        <div className="grid grid-cols-2 gap-2.5">
          {sites.map((site) => (
            <button
              key={site.id}
              type="button"
              onClick={() => setEditing(site)}
              className="flex flex-col items-start gap-2 rounded-xl border border-border bg-surface p-3 text-left transition-colors hover:border-accent/40 hover:bg-surface-secondary/50"
            >
              <span
                className="grid size-9 place-items-center rounded-lg text-[15px] font-semibold text-white"
                style={{ backgroundColor: colorFor(site.name) }}
              >
                {(site.name || '?').slice(0, 1).toUpperCase()}
              </span>
              <span className="w-full">
                <span className="block truncate text-[13px] font-medium">{site.name}</span>
                <span className="mt-0.5 block truncate text-[11px] text-muted">
                  {(site.urls ?? []).length
                    ? `${site.urls[0]}${site.urls.length > 1 ? ` +${site.urls.length - 1}` : ''}`
                    : t('siteUrls')}
                </span>
              </span>
              <span className="text-[10.5px] tabular-nums text-muted">
                {formatNumber(site.comment_count ?? 0)} {t('comment')}
              </span>
            </button>
          ))}
        </div>
      ) : (
        <EmptyState
          icon={Globe2}
          title={t('siteManage')}
          description={t('siteCount', { count: 0 })}
          action={
            <Button size="sm" variant="primary" onPress={() => setCreating(true)}>
              {t('createSite')}
            </Button>
          }
        />
      )}

      <SiteCreateDialog isOpen={creating} onClose={() => setCreating(false)} onCreated={refresh} />
      <SiteEditorDialog site={editing} onClose={() => setEditing(undefined)} onSaved={refresh} />
    </div>
  )
}
