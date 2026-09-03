import ListHTML from './list.html?raw'
import { ListLayout } from './layout'
import { createCommentNode } from './comment'
import { initListPaginatorFunc } from './page'
import type { CommentData, EventManager, DataManager, ConfigManager, List as IList } from '@/types'
import * as Utils from '@/lib/utils'
import { CommentNode } from '@/comment'
import type { Api } from '@/api'
import $t from '@/i18n'

export interface ListOptions {
  getApi: () => Api
  getEvents: () => EventManager
  getConf: () => ConfigManager
  getData: () => DataManager

  replyComment: (c: CommentData, $el: HTMLElement) => void
  editComment: (c: CommentData, $el: HTMLElement) => void
  resetEditorState: () => void
  onListGotoFirst?: () => void
}

export class List implements IList {
  private opts: ListOptions
  private $el: HTMLElement
  getEl() {
    return this.$el
  }

  private $commentsWrap: HTMLElement

  private $aiCommentsSection: HTMLElement
  private $aiCommentsWrap: HTMLElement
  getCommentsWrapEl() {
    return this.$commentsWrap
  }

  private commentNodes: CommentNode[] = []
  getCommentNodes() {
    return this.commentNodes
  }

  constructor(opts: ListOptions) {
    this.opts = opts

    // Init base element
    this.$el = Utils.createElement(ListHTML)
    this.$commentsWrap = this.$el.querySelector('.atk-list-comments-wrap')!
    this.$aiCommentsSection = this.$el.querySelector('.atk-ai-comments')!
    this.$aiCommentsWrap = this.$el.querySelector('.atk-ai-comments-wrap')!
    this.$aiCommentsSection.querySelector('.atk-ai-comments-header')!.textContent = $t('aiAssistant')

    // Init paginator
    initListPaginatorFunc({
      getList: () => this,
      ...opts,
    })

    // Bind events
    this.initCrudEvents()
  }

  getLayout({ forceFlatMode }: { forceFlatMode?: boolean } = {}) {
    return this.createLayout(this.$commentsWrap, forceFlatMode, true)
  }

  private createLayout($commentsWrap: HTMLElement, forceFlatMode?: boolean, trackNodes = true) {
    return new ListLayout({
      $commentsWrap,
      nestSortBy: this.opts.getConf().get().nestSort,
      nestMax: this.opts.getConf().get().nestMax,
      flatMode:
        typeof forceFlatMode === 'boolean'
          ? forceFlatMode
          : (this.opts.getConf().get().flatMode as boolean),
      // flatMode must be boolean because it had been handled when Artalk.init
      createCommentNode: (d, r) => {
        const node = createCommentNode({ forceFlatMode, ...this.opts }, d, r)
        if (trackNodes) this.commentNodes.push(node) // store node instance
        return node
      },
      findCommentNode: (id) => this.commentNodes.find((c) => c.getID() === id),
    })
  }

  private initCrudEvents() {
    this.opts.getEvents().on('list-load', (comments) => {
      // 导入数据
      this.getLayout().import(comments)
    })

    this.opts.getEvents().on('list-loaded', (comments) => {
      if (comments.length === 0) {
        this.commentNodes = []
        this.$commentsWrap.innerHTML = ''
      }
    })

    this.opts.getEvents().on('list-fetched', ({ data }) => {
      const aiComments = data?.ai_comments || []
      this.$aiCommentsWrap.innerHTML = ''
      this.$aiCommentsSection.hidden = aiComments.length === 0
      if (aiComments.length === 0) return

      this.createLayout(this.$aiCommentsWrap, true, false).import(aiComments)
    })

    // When comment insert
    this.opts.getEvents().on('comment-inserted', (comment) => {
      const replyComment = comment.rid
        ? this.commentNodes.find((c) => c.getID() === comment.rid)?.getData()
        : undefined
      this.getLayout().insert(comment, replyComment)
    })

    // When comment delete
    this.opts.getEvents().on('comment-deleted', (comment) => {
      const node = this.commentNodes.find((c) => c.getID() === comment.id)
      if (!node) {
        console.error(`comment node id=${comment.id} not found`)
        return
      }
      node.remove()
      this.commentNodes = this.commentNodes.filter((c) => c.getID() !== comment.id)
      // TODO: remove child nodes
    })

    // When comment update
    this.opts.getEvents().on('comment-updated', (comment) => {
      const node = this.commentNodes.find((c) => c.getID() === comment.id)
      node && node.setData(comment)
    })
  }
}
