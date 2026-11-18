import { Button, Chip, Input, Skeleton, toast } from '@heroui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ArtalkType } from 'artalk'
import { Plus, Search, Users } from 'lucide-react'
import { useState } from 'react'
import { UserEditorDialog, type EditableUser } from '@/components/UserEditorDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { Pager } from '@/components/ui/Pager'
import { Segmented } from '@/components/ui/Segmented'
import { PageHeader } from '@/components/ui/PageHeader'
import { UserAvatar } from '@/components/ui/UserAvatar'
import { useConfirm } from '@/components/ui/confirm'
import { errorMessage } from '@/lib/api'
import { getArtalk } from '@/lib/artalk'
import { usePageSize } from '@/lib/use-page-size'
import { t } from '@/lib/i18n'
import { useSession } from '@/lib/session'
import { debounce, formatNumber } from '@/lib/utils'

type UserType = 'all' | 'admin'

interface UsersResponse {
  count: number
  users: ArtalkType.UserDataForAdmin[]
}

export function UsersPage() {
  const { site } = useSession()
  const confirm = useConfirm()
  const queryClient = useQueryClient()
  const [type, setType] = useState<UserType>('all')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = usePageSize('users', 30)
  const changePageSize = (next: number) => {
    setPageSize(next)
    setPage(1)
  }
  const [editor, setEditor] = useState<{ open: boolean; user?: EditableUser }>({ open: false })

  const queryKey = ['users', site, type, page, pageSize, search]
  const { data, isPending, isFetching } = useQuery({
    queryKey,
    queryFn: async ({ signal }) => {
      const res = await getArtalk()
        .ctx.getApi()
        .users.getUsers(
          type,
          { offset: (page - 1) * pageSize, limit: pageSize, search },
          { signal },
        )
      return (res.data ?? { count: 0, users: [] }) as UsersResponse
    },
  })

  const users = data?.users ?? []
  const total = data?.count ?? 0
  const refresh = () => void queryClient.invalidateQueries({ queryKey: ['users'] })

  const remove = useMutation({
    mutationFn: (id: number) => getArtalk().ctx.getApi().users.deleteUser(id),
    onSuccess: refresh,
    onError: (error) => toast.danger(errorMessage(error)),
  })

  const runSearch = debounce((value: string) => {
    setSearch(value)
    setPage(1)
  }, 350)

  return (
    <div className="space-y-3 md:space-y-4">
      <PageHeader
        title={t('navUsers')}
        description={t('userManageDesc')}
        actions={
          <Button size="sm" variant="primary" onPress={() => setEditor({ open: true })}>
            <Plus className="size-4" aria-hidden />
            {t('create')}
          </Button>
        }
      />

      <div className="flex flex-col gap-2 md:flex-row md:items-center md:gap-3">
        <div className="relative md:flex-1">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted"
            aria-hidden
          />
          <Input
            placeholder={t('search')}
            aria-label={t('search')}
            fullWidth
            className="bg-surface pl-9"
            onChange={(event) => runSearch(event.target.value)}
          />
        </div>

        <Segmented
          items={(
            [
              ['all', 'all'],
              ['admin', 'admin'],
            ] as const
          ).map(([id, label]) => ({ id, label: t(label) }))}
          value={type}
          onChange={(next) => {
            setType(next)
            setPage(1)
          }}
        />
      </div>

      {isPending ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-16 rounded-xl" />
          ))}
        </div>
      ) : users.length ? (
        <div className="overflow-hidden rounded-xl border border-border bg-surface">
          <div className="hidden items-center gap-3 border-b border-border bg-surface-secondary/40 px-4 py-2 text-[11px] font-medium uppercase tracking-wide text-muted md:grid md:grid-cols-[minmax(0,1fr)_minmax(0,200px)_64px_auto]">
            <span>{t('user')}</span>
            <span>{t('email')}</span>
            <span className="text-right">{t('comments')}</span>
            <span className="w-[108px]" />
          </div>
          {users.map((user, index) => (
            <div
              key={user.id}
              className={`flex flex-wrap items-center gap-2.5 px-3 py-2.5 md:grid md:grid-cols-[minmax(0,1fr)_minmax(0,200px)_64px_auto] md:gap-3 md:px-4 ${
                index ? 'border-t border-border' : ''
              }`}
            >
              <div className="flex min-w-0 flex-1 items-center gap-2.5">
                <UserAvatar name={user.name} size="sm" />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-1.5">
                    <span className="truncate text-[13px] font-medium">{user.name}</span>
                    {user.badge_name ? (
                      <Chip
                        size="sm"
                        variant="soft"
                        style={{
                          color: user.badge_color,
                          backgroundColor: `${user.badge_color}1f`,
                        }}
                      >
                        {user.badge_name}
                      </Chip>
                    ) : null}
                    {!user.badge_name && user.is_admin ? (
                      <Chip size="sm" variant="soft" color="accent">
                        {t('admin')}
                      </Chip>
                    ) : null}
                    {user.is_in_conf ? (
                      <Chip size="sm" variant="soft" title={t('userInConfHint')}>
                        {t('config')}
                      </Chip>
                    ) : null}
                  </div>
                  <p className="truncate text-[11.5px] text-muted md:hidden">{user.email}</p>
                </div>
              </div>
              <span className="hidden min-w-0 truncate text-[12px] text-muted md:block">
                {user.email}
              </span>
              <span
                className="mr-1 text-[11px] tabular-nums text-muted md:mr-0 md:block md:text-right"
                title={t('comments')}
              >
                {formatNumber(user.comment_count)}
              </span>
              <div className="flex shrink-0 items-center justify-end gap-0.5">
                <Button
                  size="sm"
                  variant="ghost"
                  onPress={() =>
                    user.is_in_conf
                      ? toast.info(t('userInConfCannotEditHint'))
                      : setEditor({ open: true, user: user as unknown as EditableUser })
                  }
                >
                  {t('edit')}
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-danger"
                  isDisabled={remove.isPending}
                  onPress={async () => {
                    const ok = await confirm({
                      title: t('userDeleteConfirm', { name: user.name, email: user.email }),
                      danger: true,
                    })
                    if (!ok) return
                    remove.mutate(user.id, {
                      onSuccess: () => {
                        if (user.is_in_conf) toast.info(t('userDeleteManuallyHint'))
                      },
                    })
                  }}
                >
                  {t('delete')}
                </Button>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <EmptyState icon={Users} title={t('noUsers')} />
      )}

      <Pager
        page={page}
        pageSize={pageSize}
        total={total}
        disabled={isFetching}
        onChange={setPage}
        onPageSizeChange={changePageSize}
      />

      <UserEditorDialog
        isOpen={editor.open}
        user={editor.user}
        onClose={() => setEditor({ open: false })}
        onSaved={refresh}
      />
    </div>
  )
}
