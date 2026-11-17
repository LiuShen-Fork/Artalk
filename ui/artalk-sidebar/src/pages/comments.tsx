import { Input, Spinner, Tabs } from '@heroui/react'
import { Search } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { Pager } from '@/components/ui/Pager'
import { PageHeader } from '@/components/ui/PageHeader'
import { t, type MessageKey } from '@/lib/i18n'
import { getArtalk } from '@/lib/artalk'
import { useSession } from '@/lib/session'
import { debounce } from '@/lib/utils'

interface TabDef {
  id: string
  label: MessageKey
}

const ADMIN_TABS: TabDef[] = [
  { id: 'all', label: 'all' },
  { id: 'pending', label: 'pending' },
  { id: 'ai', label: 'aiAssistant' },
  { id: 'personal_all', label: 'personal' },
]

const USER_TABS: TabDef[] = [
  { id: 'all', label: 'all' },
  { id: 'mentions', label: 'mentions' },
  { id: 'mine', label: 'mine' },
  { id: 'pending', label: 'pending' },
]

/**
 * The comment list is the only page that renders the public Artalk widget: the
 * console keeps a headless instance for every API call, and here its list
 * element is adopted into the page so moderation actions, replies and the
 * editor all keep working exactly as on the host site.
 */
export function CommentsPage() {
  const { user, site } = useSession()
  const isAdmin = !!user?.is_admin
  const tabs = isAdmin ? ADMIN_TABS : USER_TABS

  const wrapRef = useRef<HTMLDivElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const [tab, setTab] = useState('all')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [paging, setPaging] = useState({ total: 0, pageSize: 20, page: 1 })

  // The widget's config callbacks run outside React, so the latest filter
  // values are mirrored into a ref that the fetch modifier reads on demand.
  const filter = useRef({ site, tab, search, isAdmin })
  useEffect(() => {
    filter.current = { site, tab, search, isAdmin }
  }, [site, tab, search, isAdmin])

  useEffect(() => {
    const artalk = getArtalk()
    const ctx = artalk.ctx
    const previous = { ...artalk.getConf() }
    const listEl = ctx.inject('list').getEl()

    const onCommentRendered = (comment: any) => {
      const pageURL = comment.getData?.().page_url
      if (pageURL) comment.getRender().setOpenURL(`${pageURL}#atk-comment-${comment.getID()}`)
    }
    const onLoaded = () => {
      const data = ctx.inject('data')
      const last = data.getListLastFetch()
      if (last?.data) {
        setPaging({
          total: last.params.flatMode ? last.data.count : last.data.roots_count,
          pageSize: last.params.limit,
          page: Math.floor(last.params.offset / last.params.limit) + 1,
        })
      }
      setLoading(false)
    }
    const onFetch = () => setLoading(true)
    const onFetched = () => setLoading(false)
    const onFailed = () => setLoading(false)

    ctx.on('comment-rendered', onCommentRendered)
    ctx.on('list-loaded', onLoaded)
    ctx.on('list-fetch', onFetch)
    ctx.on('list-fetched', onFetched)
    ctx.on('list-failed', onFailed)

    ctx.updateConf({
      noComment: `<div class="py-8 text-center text-[13px] text-muted">${t('noContent')}</div>`,
      listFetchParamsModifier: (params: any) => {
        const current = filter.current
        params.site_name = current.site
        params.scope = current.isAdmin && current.tab !== 'personal_all' ? 'site' : 'user'
        params.type = current.tab === 'personal_all' ? 'all' : current.tab
        // Keep AI comments in the normal list; the AI tab filters server-side.
        params.split_ai = false
        if (current.search) params.search = current.search
      },
      scrollRelativeTo: () => wrapRef.current ?? document.body,
    })

    // The widget's own header and footer are replaced by the console chrome.
    listEl.querySelector<HTMLElement>('.atk-list-header')?.style.setProperty('display', 'none')
    listEl.querySelector<HTMLElement>('.atk-list-footer')?.style.setProperty('display', 'none')
    listRef.current?.append(listEl)
    artalk.reload()

    return () => {
      ctx.off('comment-rendered', onCommentRendered)
      ctx.off('list-loaded', onLoaded)
      ctx.off('list-fetch', onFetch)
      ctx.off('list-fetched', onFetched)
      ctx.off('list-failed', onFailed)
      ctx.updateConf({
        listFetchParamsModifier: previous.listFetchParamsModifier,
        scrollRelativeTo: previous.scrollRelativeTo,
      })
      ctx.editorResetState()
      listEl.remove()
    }
    // The widget is adopted once; later filter changes go through the effects
    // below instead of tearing the list down and rebuilding it.
  }, [])

  useEffect(() => {
    if (!listRef.current?.hasChildNodes()) return
    getArtalk().ctx.fetch({ offset: 0 })
  }, [tab])

  useEffect(() => {
    if (!listRef.current?.hasChildNodes()) return
    getArtalk().reload()
  }, [site])

  const runSearch = useRef(
    debounce((value: string) => {
      setSearch(value)
      getArtalk().reload()
    }, 400),
  ).current

  return (
    <div className="space-y-3">
      <PageHeader title={t('comment')} description={t('commentManageDesc')} />
      <div className="relative">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted"
          aria-hidden
        />
        <Input
          value={search}
          onChange={(event) => runSearch(event.target.value)}
          placeholder={t('searchHint')}
          aria-label={t('searchHint')}
          fullWidth
          className="bg-surface pl-9"
        />
      </div>

      <Tabs selectedKey={tab} onSelectionChange={(key) => setTab(String(key))}>
        <Tabs.ListContainer className="w-full">
          <Tabs.List className="w-full justify-between">
            {tabs.map((item) => (
              <Tabs.Tab key={item.id} id={item.id} className="flex-1 text-[12.5px]">
                {t(item.label)}
              </Tabs.Tab>
            ))}
          </Tabs.List>
        </Tabs.ListContainer>
      </Tabs>

      <div ref={wrapRef} className="relative min-h-40">
        {loading ? (
          <div className="pointer-events-none absolute inset-x-0 top-0 flex justify-center pt-2">
            <Spinner size="sm" />
          </div>
        ) : null}
        <div ref={listRef} className="atk-comments" />
      </div>

      <Pager
        page={paging.page}
        pageSize={paging.pageSize}
        total={paging.total}
        disabled={loading}
        onChange={(next) => {
          getArtalk().ctx.editorResetState()
          getArtalk().ctx.fetch({
            offset: (next - 1) * paging.pageSize,
            onSuccess: () => getArtalk().ctx.listGotoFirst(),
          })
        }}
      />
    </div>
  )
}
