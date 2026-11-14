// @vitest-environment jsdom
import { expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import Pagination from '../components/Pagination.vue'

interface PaginationVm {
  prev: () => void
  next: () => void
  reset: () => void
}

function mountPagination(props: Record<string, unknown>) {
  const change = vi.fn()
  const container = document.createElement('div')
  const app = createApp(Pagination, { pageSize: 20, onChange: change, ...props })
  const vm = app.mount(container) as unknown as PaginationVm
  return { change, app, container, vm }
}

function pageButtons(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLButtonElement>('.page-btn'))
}

function activePage(container: HTMLElement) {
  return container.querySelector('.page-btn.active')?.textContent?.trim()
}

it('hides the pager when everything fits on a single page', () => {
  const { app, container } = mountPagination({ total: 10 })
  try {
    expect(container.querySelector('.atk-page-nav')).toBeNull()
  } finally {
    app.unmount()
  }
})

it('reports the offset of the clicked page and keeps one active page', async () => {
  const { change, app, container } = mountPagination({ total: 100 })
  try {
    expect(activePage(container)).toBe('1')
    const third = pageButtons(container).find((btn) => btn.textContent?.trim() === '3')
    third!.dispatchEvent(new MouseEvent('click'))
    await nextTick()
    expect(change).toHaveBeenCalledExactlyOnceWith(40)
    expect(activePage(container)).toBe('3')
  } finally {
    app.unmount()
  }
})

it('keeps a compact window with ellipsis around the page supplied by the owner', async () => {
  const page = ref(10)
  const container = document.createElement('div')
  const app = createApp({ render: () => h(Pagination, { total: 400, pageSize: 20, page: page.value }) })
  try {
    app.mount(container)
    await nextTick()
    expect(activePage(container)).toBe('10')
    expect(container.textContent?.match(/…/g)).toHaveLength(2)
    expect(pageButtons(container).map((btn) => btn.textContent?.trim())).toEqual([
      '1',
      '9',
      '10',
      '11',
      '12',
      '20',
    ])

    page.value = 20
    await nextTick()
    expect(activePage(container)).toBe('20')
  } finally {
    app.unmount()
  }
})

it('clamps navigation at the edges and resets without emitting', async () => {
  const { change, app, container, vm } = mountPagination({ total: 60 })
  try {
    const [prevBtn, nextBtn] = Array.from(container.querySelectorAll<HTMLButtonElement>('.page-arrow'))
    expect(prevBtn.disabled).toBe(true)
    expect(nextBtn.disabled).toBe(false)

    nextBtn.dispatchEvent(new MouseEvent('click'))
    await nextTick()
    expect(change).toHaveBeenLastCalledWith(20)
    expect(activePage(container)).toBe('2')

    nextBtn.dispatchEvent(new MouseEvent('click'))
    await nextTick()
    expect(activePage(container)).toBe('3')
    expect(nextBtn.disabled).toBe(true)

    change.mockClear()
    vm.reset()
    await nextTick()
    expect(activePage(container)).toBe('1')
    expect(change).not.toHaveBeenCalled()
  } finally {
    app.unmount()
  }
})
