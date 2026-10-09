import { inject, provide, ref, watch, type InjectionKey, type Ref } from "vue"

const draftsKey: InjectionKey<Map<string, Ref>> = Symbol("instrument drafts")
const authorityKey: InjectionKey<Ref<string>> = Symbol("instrument authority")
const errorKey: InjectionKey<(message: string) => void> = Symbol("instrument error")

export function provideInstrumentSession(authority: Ref<string>, reportError: (message: string) => void): void {
  provide(draftsKey, new Map())
  provide(authorityKey, authority)
  provide(errorKey, reportError)
}

// Only desired values belong here. Review authority and DOM references remain local.
export function useInstrumentDraft<T>(key: string, initial: T): Ref<T> {
  const drafts = inject(draftsKey, undefined)
  if (!drafts) return ref(initial) as Ref<T>
  if (!drafts.has(key)) drafts.set(key, ref(initial))
  return drafts.get(key) as Ref<T>
}

export function invalidateInstrumentReview(invalidate: () => void): void {
  const authority = inject(authorityKey, undefined)
  if (authority) watch(authority, invalidate, { flush: "sync" })
}

export function useInstrumentError(): (message: string) => void {
  return inject(errorKey, () => {})
}
