import type { ObjectDirective } from "vue"

const focusable = "button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href], [tabindex]:not([tabindex='-1'])"
const cleanups = new WeakMap<HTMLElement, () => void>()

export const vOwnedModal: ObjectDirective<HTMLElement, () => void> = {
  mounted(element, binding) {
    const previous = document.activeElement as HTMLElement | null
    const background = new Map<HTMLElement, boolean>()
    let branch: HTMLElement = element
    while (branch.parentElement) {
      for (const sibling of branch.parentElement.children) {
        if (sibling !== branch && sibling instanceof HTMLElement) {
          background.set(sibling, sibling.inert)
          sibling.inert = true
        }
      }
      branch = branch.parentElement
      if (branch === document.body) break
    }
    element.tabIndex = -1
    const controls = () => [...element.querySelectorAll<HTMLElement>(focusable)].filter((control) => !control.closest("[hidden], [inert]") && control.getAttribute("aria-hidden") !== "true")
    const keydown = (event: KeyboardEvent) => {
      if (event.key === "F5" || ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k")) {
        event.preventDefault()
        event.stopImmediatePropagation()
      } else if (event.key === "Escape") {
        event.preventDefault()
        event.stopImmediatePropagation()
        binding.value()
      } else if (event.key === "Tab") {
        const targets = controls()
        const first = targets[0] ?? element
        const last = targets.at(-1) ?? element
        if (!element.contains(document.activeElement) || (event.shiftKey ? document.activeElement === first : document.activeElement === last)) {
          event.preventDefault()
          ;(event.shiftKey ? last : first).focus()
        }
      }
    }
    const focusin = (event: FocusEvent) => {
      if (!element.contains(event.target as Node)) (controls()[0] ?? element).focus()
    }
    document.addEventListener("keydown", keydown, true)
    document.addEventListener("focusin", focusin)
    ;(controls()[0] ?? element).focus()
    cleanups.set(element, () => {
      document.removeEventListener("keydown", keydown, true)
      document.removeEventListener("focusin", focusin)
      for (const [node, inert] of background) node.inert = inert
      if (previous?.isConnected) previous.focus()
    })
  },
  beforeUnmount(element) {
    cleanups.get(element)?.()
    cleanups.delete(element)
  },
}
