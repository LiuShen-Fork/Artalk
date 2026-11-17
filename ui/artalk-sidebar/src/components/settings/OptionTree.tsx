import { Button, Input, ListBox, Select as HeroSelect, Switch, TextArea } from '@heroui/react'
import { Eye, EyeOff, Minus, Plus } from 'lucide-react'
import { useState } from 'react'
import settings, {
  KEYWORD_FILE_SEPARATOR_PATH,
  formatOptionValue,
  patchOptionValue,
  type OptionNode,
} from '@/lib/settings'
import { isSensitiveConfigPath } from '@/lib/settings-sensitive'
import { t } from '@/lib/i18n'
import { cn } from '@/lib/utils'

const SEPARATOR_SUGGESTIONS = ['\\n', '\\r\\n', '\\t', '|']

const envName = (path: string) => `ATK_${path.replace(/\./g, '_').toUpperCase()}`

/**
 * The config editor is generated from the commented YAML template: every leaf
 * becomes a control chosen by its type, so a new config field needs no UI code.
 */
export function OptionItem({ node }: { node: OptionNode }) {
  const isThinkingMode = node.path === 'moderator.ai.disable_thinking'
  const isSensitive = isSensitiveConfigPath(node.path)
  const isSeparator = node.path === KEYWORD_FILE_SEPARATOR_PATH

  const [value, setValue] = useState<any>(() =>
    formatOptionValue(settings.get().getCustom(node.path) ?? node.default ?? '', node),
  )
  const [revealed, setRevealed] = useState(!isSensitive)
  const disabled = !!settings.get().getEnvByPath(node.path)

  const commit = (next: any) => {
    setValue(next)
    settings.get().setCustom(node.path, patchOptionValue(next, node))
  }

  const label = (
    <div className="min-w-0">
      <p className="text-[12.5px] font-medium leading-snug" title={envName(node.path)}>
        {node.title}
      </p>
      {node.subTitle ? (
        <p className="mt-0.5 text-[11px] leading-snug text-muted">{node.subTitle}</p>
      ) : null}
    </div>
  )

  if (node.type === 'array') {
    return (
      <div className="space-y-1.5 py-2">
        {label}
        <OptionArray node={node} disabled={disabled} />
      </div>
    )
  }

  if (node.type === 'boolean') {
    return (
      <div className="flex items-center justify-between gap-3 py-2">
        {label}
        <Switch
          isSelected={isThinkingMode ? value !== true : value === true || value === 'true'}
          isDisabled={disabled}
          onChange={(checked) => commit(isThinkingMode ? !checked : checked)}
        >
          <Switch.Content>
            <Switch.Control>
              <Switch.Thumb />
            </Switch.Control>
          </Switch.Content>
        </Switch>
      </div>
    )
  }

  return (
    <div className="space-y-1.5 py-2">
      {label}
      {disabled ? (
        <p className="rounded-md bg-default/60 px-2 py-1 text-[11px] text-muted">
          {t('envVarControlHint', { key: envName(node.path) })}
        </p>
      ) : null}

      {node.selector ? (
        <HeroSelect
          selectedKey={String(value)}
          isDisabled={disabled}
          onSelectionChange={(key) => commit(String(key))}
        >
          <HeroSelect.Trigger className="w-full">
            <HeroSelect.Value />
            <HeroSelect.Indicator />
          </HeroSelect.Trigger>
          <HeroSelect.Popover>
            <ListBox>
              {node.selector.map((option) => (
                <ListBox.Item key={option} id={option} textValue={option}>
                  {option}
                </ListBox.Item>
              ))}
            </ListBox>
          </HeroSelect.Popover>
        </HeroSelect>
      ) : node.control === 'textarea' ? (
        <TextArea
          value={String(value ?? '')}
          disabled={disabled}
          rows={6}
          placeholder={node.placeholder}
          className="text-[12px] leading-relaxed"
          onChange={(event) => setValue(event.target.value)}
          onBlur={() => commit(value)}
        />
      ) : (
        <div className="relative">
          <Input
            type={revealed ? 'text' : 'password'}
            value={String(value ?? '')}
            disabled={disabled}
            placeholder={node.placeholder}
            list={isSeparator ? 'atk-keyword-separator' : undefined}
            className={cn('text-[12px]', isSensitive && 'pr-9')}
            onChange={(event) => setValue(event.target.value)}
            onBlur={() => commit(value)}
          />
          {isSeparator ? (
            <datalist id="atk-keyword-separator">
              {SEPARATOR_SUGGESTIONS.map((item) => (
                <option key={item} value={item} />
              ))}
            </datalist>
          ) : null}
          {isSensitive ? (
            <button
              type="button"
              aria-label={revealed ? t('hide') : t('show')}
              onClick={() => setRevealed((prev) => !prev)}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted hover:text-foreground"
            >
              {revealed ? (
                <EyeOff className="size-3.5" aria-hidden />
              ) : (
                <Eye className="size-3.5" aria-hidden />
              )}
            </button>
          ) : null}
        </div>
      )}
    </div>
  )
}

function OptionArray({ node, disabled }: { node: OptionNode; disabled: boolean }) {
  const initial = (): string[] => {
    const value = settings.get().getCustom(node.path)
    if (value && typeof value === 'object' && typeof value.toJSON === 'function') {
      return (value.toJSON() as unknown[]).map(String)
    }
    if (typeof value === 'string') return value.split(' ')
    return []
  }
  const [items, setItems] = useState<string[]>(initial)

  const save = (next: string[]) => {
    setItems(next)
    settings.get().setCustom(node.path, patchOptionValue(next, node))
  }

  return (
    <div className="space-y-1.5">
      {items.map((item, index) => (
        <div key={index} className="flex items-center gap-1.5">
          <Input
            value={item}
            disabled={disabled}
            className="text-[12px]"
            onChange={(event) => {
              const next = [...items]
              next[index] = event.target.value
              save(next)
            }}
          />
          {!disabled ? (
            <Button
              size="sm"
              variant="ghost"
              isIconOnly
              aria-label={t('delete')}
              onPress={() => save(items.filter((_, i) => i !== index))}
            >
              <Minus className="size-3.5" aria-hidden />
            </Button>
          ) : null}
        </div>
      ))}
      {!disabled ? (
        <Button size="sm" variant="ghost" onPress={() => save([...items, ''])}>
          <Plus className="size-3.5" aria-hidden />
          {t('add')}
        </Button>
      ) : null}
    </div>
  )
}

export function OptionGroup({ node, depth = 0 }: { node: OptionNode; depth?: number }) {
  if (node.name === 'admin_users') return null

  if (!node.items?.length) return <OptionItem node={node} />

  return (
    <section className={cn('space-y-0.5', depth === 0 ? '' : 'border-l border-border pl-3')}>
      {node.title ? (
        <header className={cn('pb-1', depth === 0 ? 'pt-1' : 'pt-2')}>
          <h3
            className={cn(
              'font-semibold tracking-[-0.01em]',
              depth === 0 ? 'text-[13px]' : 'text-[12.5px]',
            )}
          >
            {node.title}
          </h3>
          {node.subTitle ? <p className="mt-0.5 text-[11px] text-muted">{node.subTitle}</p> : null}
        </header>
      ) : null}
      <div className="divide-y divide-border/60">
        {node.items.map((child) =>
          child.items?.length ? (
            <div key={child.path} className="py-1">
              <OptionGroup node={child} depth={depth + 1} />
            </div>
          ) : (
            <OptionItem key={child.path} node={child} />
          ),
        )}
      </div>
    </section>
  )
}
