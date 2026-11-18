import { Input, Spinner } from '@heroui/react'
import { Search } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { Pager } from '@/components/ui/Pager'
import { Segmented } from '@/components/ui/Segmented'
import { PageHeader } from '@/components/ui/PageHeader'
import { t, type MessageKey } from '@/lib/i18n'
import { getArtalk } from '@/lib/artalk'
import { useSession } from '@/lib/session'
import { usePageSize } from '@/lib/use-page-size'
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
  const { user, site, view } = useSession()
  const isAdmin = !!user?.is_admin
  // "My pages" mode narrows the same list to what this account wrote.
  const manage = isAdmin && view === 'admin'
  const tabs = manage ? ADMIN_TABS : USER_TABS

  const wrapRef = useRef<HTMLDivElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const [tab, setTab] = useState('all')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [pageSize, setPageSize] = usePageSize('comments', 20)
  const [paging, setPaging] = useState({ total: 0, page: 1 })

  // The widget reads its page size from its own config, which its fetch
  // callbacks cannot reach from React, so the current value is mirrored here.
  const pageSizeRef = useRef(pageSize)

  // The widget's config callbacks run outside React, so the latest filter
  // values are mirrored into a ref that the fetch modifier reads on demand.
  const filter = useRef({ site, tab, search, manage })
  useEffect(() => {
    filter.current = { site, tab, search, manage }
  }, [site, tab, search, manage])

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
      pagination: { pageSize: pageSizeRef.current },
      noComment: `<div class="py-8 text-center text-[13px] text-muted">${t('noContent')}</div>`,
      listFetchParamsModifier: (params: any) => {
        const current = filter.current
        params.site_name = current.site
        params.scope = current.manage && current.tab !== 'personal_all' ? 'site' : 'user'
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
        pagination: previous.pagination,
        listFetchParamsModifier: previous.listFetchParamsModifier,
        scrollRelativeTo: previous.scrollRelativeTo,
      })
      ctx.editorResetState()
      listEl.remove()
    }
    // The widget is adopted once; later filter changes go through the effects
    // below instead of tearing the list down and rebuilding it.
  }, [])

  // Switching between the admin console and "my pages" swaps the whole tab
  // set, so an unknown tab falls back to the first one instead of a dead list.
  useEffect(() => {
    if (!tabs.some((item) => item.id === tab)) setTab(tabs[0].id)
  }, [tabs, tab])

  useEffect(() => {
    if (!listRef.current?.hasChildNodes()) return
    getArtalk().ctx.fetch({ offset: 0 })
  }, [tab])

  useEffect(() => {
    if (!listRef.current?.hasChildNodes()) return
    getArtalk().reload()
  }, [site])

  // Page size is a widget config value, so the list has to be refetched from
  // the first page after it changes. The first run only records the stored
  // preference: the mount effect above already applied it.
  useEffect(() => {
    if (pageSizeRef.current === pageSize) return
    pageSizeRef.current = pageSize
    if (!listRef.current?.hasChildNodes()) return
    const ctx = getArtalk().ctx
    ctx.updateConf({ pagination: { pageSize } })
    ctx.fetch({ offset: 0 })
  }, [pageSize])

  const runSearch = useRef(
    debounce((value: string) => {
      setSearch(value)
      getArtalk().reload()
    }, 400),
  ).current

  const goTo = (next: number) => {
    const ctx = getArtalk().ctx
    ctx.editorResetState()
    ctx.fetch({
      offset: (next - 1) * pageSize,
      onSuccess: () => ctx.listGotoFirst(),
    })
  }

  // One pager, under the list, where a reader who just finished a page ends up.
  // The widget's own pager is hidden in styles/index.css so the console only
  // ever shows this one.
  const pager = (
    <Pager
      page={paging.page}
      pageSize={pageSize}
      total={paging.total}
      disabled={loading}
      onChange={goTo}
      onPageSizeChange={setPageSize}
    />
  )

  return (
    <div className="space-y-3 md:space-y-4">
      <PageHeader
        title={manage ? t('navComments') : t('navMyComments')}
        description={manage ? t('commentManageDesc') : t('myCommentsDesc')}
      />

      <div className="flex flex-col gap-2 md:flex-row md:items-center md:gap-3">
        <div className="relative md:flex-1">
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

        <Segmented
          items={tabs.map((item) => ({ id: item.id, label: t(item.label) }))}
          value={tab}
          onChange={setTab}
        />
      </div>

      <div className="space-y-3">
        <div ref={wrapRef} className="relative min-h-40">
          {loading ? (
            <div className="pointer-events-none absolute inset-x-0 top-0 flex justify-center pt-2">
              <Spinner size="sm" />
            </div>
          ) : null}
          <div ref={listRef} className="atk-comments" />
        </div>

        {pager}
      </div>
    </div>
  )
}
