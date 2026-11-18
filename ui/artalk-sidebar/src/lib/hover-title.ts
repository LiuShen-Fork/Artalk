/**
 * Native hover hint for icon-only controls.
 *
 * HeroUI buttons are react-aria buttons, and react-aria keeps `title` out of
 * the DOM props it forwards, so a plain `title` prop is dropped in the browser.
 * Set it on the rendered node instead; the accessible name still comes from
 * `aria-label`.
 */
export const hoverTitle = (title: string) => ({
  ref: (el: HTMLElement | null) => {
    if (el) el.title = title
  },
})
