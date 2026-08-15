<script setup lang="ts">
import type { ArtalkType } from 'artalk'
import { storeToRefs } from 'pinia'
import { useNavStore } from '../stores/nav'
import { artalk } from '../global'
import Pagination from '../components/Pagination.vue'

const nav = useNavStore()
const { curtTab } = storeToRefs(nav)
const users = ref<ArtalkType.UserDataForAdmin[]>([])
const { t } = useI18n()

const pageSize = ref(30)
const pageTotal = ref(0)
const pagination = ref<InstanceType<typeof Pagination>>()
const curtType = ref<'all' | 'admin' | 'in_conf' | undefined>('all')

const userEditorState = reactive({
  show: false,
  user: undefined as ArtalkType.UserDataForAdmin | undefined,
})
const search = ref('')

watch(curtTab, (tab) => {
  if (tab === 'create') {
    createUser()
  } else {
    curtType.value = tab as any
    fetchUsers(0)
    closeUserEditor()
  }
})

onMounted(() => {
  fetchUsers(0)

  nav.updateTabs(
    {
      all: 'all',
      admin: 'admin',
      create: 'create',
    },
    'all',
  )

  // Users search
  nav.enableSearch(
    (value: string) => {
      search.value = value
      fetchUsers(0)
    },
    () => {
      if (search.value === '') return
      search.value = ''
      fetchUsers(0)
    },
  )
})

watch(
  () => userEditorState.show,
  () => nav.scrollPageToTop(),
)

function fetchUsers(offset: number) {
  if (offset === 0) pagination.value?.reset()
  nav.setPageLoading(true)
  artalk?.ctx
    .getApi()
    .users.getUsers(curtType.value, {
      offset,
      limit: pageSize.value,
      search: search.value,
    })
    .then((res) => {
      pageTotal.value = res.data.count
      users.value = res.data.users
      nav.scrollPageToTop()
    })
    .finally(() => {
      nav.setPageLoading(false)
    })
}

function onChangePage(offset: number) {
  fetchUsers(offset)
}

function editUser(user: ArtalkType.UserDataForAdmin) {
  if (user.is_in_conf) {
    alert(t('userInConfCannotEditHint'))
    return
  }

  userEditorState.show = true
  userEditorState.user = user
}

function createUser() {
  userEditorState.show = true
  userEditorState.user = undefined
}

function updateUser(user: ArtalkType.UserDataForAdmin) {
  const index = users.value.findIndex((u) => u.id === user.id)
  if (index != -1) {
    // Edit user
    const orgUser = users.value[index]
    Object.keys(user).forEach((key) => {
      ;(orgUser as any)[key] = (user as any)[key]
    })
  } else {
    // Create user
    fetchUsers(0)
  }

  closeUserEditor()
}

function closeUserEditor() {
  userEditorState.show = false
  userEditorState.user = undefined
}

function delUser(user: ArtalkType.UserDataForAdmin) {
  if (
    window.confirm(
      t('userDeleteConfirm', {
        name: user.name,
        email: user.email,
      }),
    )
  ) {
    artalk!.ctx
      .getApi()
      .users.deleteUser(user.id)
      .then(() => {
        const index = users.value.findIndex((u) => u.id === user.id)
        users.value.splice(index, 1)

        if (user.is_in_conf) {
          alert(t('userDeleteManuallyHint'))
        }
      })
      .catch((e: ArtalkType.FetchError) => {
        alert(e.message)
      })
  }
}
</script>

<template>
  <div class="user-list-wrap admin-page">
    <div class="user-list admin-panel">
      <div v-for="user in users" :key="user.id" class="user-item">
        <div class="user-main">
          <div class="title">
            {{ user.name }}
            <span class="badge-grp">
              <span
                v-if="user.badge_name"
                class="badge"
                :style="{ backgroundColor: user.badge_color }"
              >
                {{ user.badge_name }}
              </span>
              <span v-else-if="user.is_admin" class="badge admin" :title="t('userAdminHint')">
                {{ t('Admin') }}
              </span>
              <span v-if="user.is_in_conf" class="badge in-conf" :title="t('userInConfHint')">
                {{ t('Config') }}
              </span>
            </span>
          </div>
          <div class="sub">{{ user.email }}</div>
        </div>
        <div class="user-actions">
          <span @click="editUser(user)">{{ t('edit') }}</span>
          <span>{{ t('comment') }} ({{ user.comment_count }})</span>
          <span @click="delUser(user)">{{ t('delete') }}</span>
        </div>
      </div>
    </div>
    <Pagination
      ref="pagination"
      :page-size="pageSize"
      :total="pageTotal"
      :disabled="nav.isPageLoading"
      @change="onChangePage"
    />

    <UserEditor
      v-if="userEditorState.show"
      :user="userEditorState.user"
      @update="updateUser"
      @close="closeUserEditor"
    />
  </div>
</template>

<style scoped lang="scss">
.user-list-wrap {
  .user-list {
    .user-item {
      padding: 15px 30px;

      &:not(:last-child) {
        border-bottom: 1px solid var(--at-color-border);
      }

      .user-main {
        .title {
          display: flex;
          flex-direction: row;
          align-items: center;
          color: var(--at-color-deep);
          font-size: 20px;

          .badge-grp {
            display: flex;
            margin-left: 5px;
          }

          .badge {
            margin-left: 6px;
            font-size: 13px;
            color: var(--at-color-meta);
            background: var(--at-color-bg-grey);
            padding: 0 6px;
            line-height: 17px;
            border-radius: 2px;
            color: #fff;

            &.admin {
              background: var(--atk-admin-terracotta);
            }

            &.in-conf {
              background: #89b1a5;
            }
          }
        }

        .sub {
          color: var(--at-color-sub);
          font-size: 15px;
          margin-top: 5px;
        }
      }

      .user-actions {
        margin-top: 10px;

        & > span {
          cursor: pointer;
          color: var(--at-color-meta);
          font-size: 13px;

          &:not(:last-child) {
            margin-right: 16px;
          }

          &:hover {
            color: var(--at-color-deep);
          }
        }
      }
    }
  }
}

.user-list-wrap {
  .user-list {
    overflow: hidden;
    border-color: var(--atk-admin-border);
    background: var(--atk-admin-surface);
    border-radius: var(--atk-admin-radius);
    box-shadow: 0 2px 8px rgba(72, 60, 46, 0.04);

    .user-item {
      padding: 18px 20px;
      border-color: var(--atk-admin-border);

      .user-main {
        .title {
          color: var(--atk-admin-ink);
          font-size: 18px;

          .badge {
            border-radius: 999px;
            background: var(--atk-admin-sage);

            &.admin {
              background: var(--atk-admin-terracotta);
            }

            &.in-conf {
              background: var(--atk-admin-sage);
            }
          }
        }

        .sub {
          color: var(--atk-admin-subtle);
        }
      }

      .user-actions > span {
        color: var(--atk-admin-subtle);

        &:hover {
          color: var(--atk-admin-sage);
        }
      }
    }
  }
}

@media (max-width: 700px) {
  .user-list-wrap .user-list .user-item {
    padding: 16px;
  }
}
</style>
