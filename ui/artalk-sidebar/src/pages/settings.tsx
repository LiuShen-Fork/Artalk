import { Button, Spinner, toast } from '@heroui/react'
import { Save } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import YAML from 'yaml'
import { PageHeader } from '@/components/ui/PageHeader'
import { SectionNav } from '@/components/ui/SectionNav'
import { OptionGroup } from '@/components/settings/OptionTree'
import { getArtalk } from '@/lib/artalk'
import { errorMessage } from '@/lib/api'
import { t } from '@/lib/i18n'
import settings, { type OptionNode } from '@/lib/settings'

const HIDDEN_ROOT_NODES = new Set(['admin_users'])

/**
 * Root entries without children are flattened into one "general" section so the
 * index does not fill up with single-field categories such as `app_key`.
 */
function buildRootGroups(tree: OptionNode): OptionNode[] {
  const visible = (tree.items ?? []).filter((node) => !HIDDEN_ROOT_NODES.has(node.name))
  const leaves = visible
    .filter((node) => !node.items?.length)
    .map((node) => ({ ...node, level: 2 }))

  const groups = visible.filter((node) => !!node.items?.length)

  if (leaves.length) {
    groups.unshift({
      name: '__basic__',
      path: '__basic__',
      level: 1,
      type: 'object',
      title: t('config'),
      items: leaves,
    })
  }

  return groups
}

export function SettingsPage() {
  const [tree, setTree] = useState<OptionNode | null>(null)
  const [selected, setSelected] = useState('')
  const [saving, setSaving] = useState(false)

  const groups = useMemo(() => (tree ? buildRootGroups(tree) : []), [tree])
  const active = groups.find((group) => group.path === selected) ?? groups[0]

  useEffect(() => {
    let alive = true

    Promise.all([
      getArtalk().ctx.getApi().settings.getSettingsTemplate('zh-CN'),
      getArtalk().ctx.getApi().settings.getSettings(),
    ])
      .then(([template, custom]) => {
        if (!alive) return
        const document = YAML.parseDocument(template.data.yaml)
        const parsed = settings.init(document).getTree()
        settings.get().setCustoms(custom.data.yaml)
        settings.get().setEnvs(custom.data.envs)
        setTree(parsed)
        const first = buildRootGroups(parsed)[0]
        if (first) setSelected(first.path)
      })
      .catch((error) => toast.danger(errorMessage(error)))

    return () => {
      alive = false
    }
  }, [])

  const save = async () => {
    let yaml: string
    try {
      yaml = settings.get().getCustoms()?.toString() ?? ''
    } catch (error) {
      toast.danger(`YAML export error: ${String(error)}`)
      return
    }
    if (!yaml) {
      toast.danger(t('settingSaveFailed'))
      return
    }

    setSaving(true)
    try {
      await getArtalk().ctx.getApi().settings.applySettings({ yaml })
      toast.success(t('settingSaved'))
    } catch (error) {
      toast.danger(`${t('settingSaveFailed')}: ${errorMessage(error)}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-3 md:space-y-4 xl:max-w-[1120px]">
      <PageHeader
        title={t('settings')}
        description={t('settingNotice')}
        actions={
          <Button
            size="sm"
            variant="primary"
            isDisabled={!tree || saving}
            onPress={() => void save()}
          >
            {saving ? <Spinner size="sm" /> : <Save className="size-3.5" aria-hidden />}
            {t('apply')}
          </Button>
        }
      />

      {!tree ? (
        <div className="flex items-center justify-center gap-2 py-16 text-[12.5px] text-muted">
          <Spinner size="sm" />
          {t('refreshing')}
        </div>
      ) : (
        <div className="md:flex md:items-start md:gap-6">
          <SectionNav
            items={groups.map((group) => ({ id: group.path, label: group.title }))}
            value={active?.path ?? ''}
            onChange={setSelected}
          />

          {/* Fields stay readable on a wide screen instead of stretching the
              full 1000px+ of the content column. */}
          <div className="min-w-0 flex-1">
            {active ? (
              <div
                key={active.path}
                className="rounded-xl border border-border bg-surface px-3 py-2 md:px-5 md:py-3"
              >
                <OptionGroup node={active} />
              </div>
            ) : null}
          </div>
        </div>
      )}
    </div>
  )
}
