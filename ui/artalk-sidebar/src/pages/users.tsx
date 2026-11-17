import { Button, Chip, Input, Skeleton, Tabs, toast } from '@heroui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ArtalkType } from 'artalk'
import { Plus, Search, Users } from 'lucide-react'
import { useState } from 'react'
import { UserEditorDialog, type EditableUser } from '@/components/UserEditorDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { Pager } from '@/components/ui/Pager'
import { PageHeader } from '@/components/ui/PageHeader'
import { UserAvatar } from '@/components/ui/UserAvatar'
import { useConfirm } from '@/components/ui/confirm'
import { errorMessage } from '@/lib/api'
import { getArtalk } from '@/lib/artalk'
import { t } from '@/lib/i18n'
import { useSession } from '@/lib/session'
import { debounce, formatNumber } from '@/lib/utils'

const PAGE_SIZE = 30
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
  const [editor, setEditor] = useState<{ open: boolean; user?: EditableUser }>({ open: false })

  const queryKey = ['users', site, type, page, search]
  const { data, isPending, isFetching } = useQuery({
    queryKey,
    queryFn: async ({ signal }) => {
      const res = await getArtalk()
        .ctx.getApi()
        .users.getUsers(
          type,
          { offset: (page - 1) * PAGE_SIZE, limit: PAGE_SIZE, search },
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
    <div className="space-y-3">
      <PageHeader
        title={t('user')}
        description={t('userManageDesc')}
        actions={
          <Button size="sm" variant="primary" onPress={() => setEditor({ open: true })}>
            <Plus className="size-4" aria-hidden />
            {t('create')}
          </Button>
        }
      />

      <div className="relative">
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

      <Tabs
        selectedKey={type}
        onSelectionChange={(key) => {
          setType(String(key) as UserType)
          setPage(1)
        }}
      >
        <Tabs.ListContainer className="w-full">
          <Tabs.List className="w-full justify-between">
            {(
              [
                ['all', 'all'],
                ['admin', 'admin'],
              ] as const
            ).map(([id, label]) => (
              <Tabs.Tab key={id} id={id} className="flex-1 text-[12.5px]">
                {t(label)}
              </Tabs.Tab>
            ))}
          </Tabs.List>
        </Tabs.ListContainer>
      </Tabs>

      {isPending ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-16 rounded-xl" />
          ))}
        </div>
      ) : users.length ? (
        <div className="overflow-hidden rounded-xl border border-border bg-surface">
          {users.map((user, index) => (
            <div
              key={user.id}
              className={`flex items-center gap-2.5 px-3 py-2.5 ${
                index ? 'border-t border-border' : ''
              }`}
            >
              <UserAvatar name={user.name} size="sm" />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5">
                  <span className="truncate text-[13px] font-medium">{user.name}</span>
                  {user.badge_name ? (
                    <Chip
                      size="sm"
                      variant="soft"
                      style={{ color: user.badge_color, backgroundColor: `${user.badge_color}1f` }}
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
                <p className="truncate text-[11.5px] text-muted">{user.email}</p>
              </div>
              <div className="flex shrink-0 items-center gap-0.5">
                <span className="mr-1 text-[11px] tabular-nums text-muted" title={t('comments')}>
                  {formatNumber(user.comment_count)}
                </span>
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
        pageSize={PAGE_SIZE}
        total={total}
        disabled={isFetching}
        onChange={setPage}
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
