type LocaleMessages = Record<string, unknown>

const isMessages = (value: unknown): value is LocaleMessages =>
  value !== null && typeof value === 'object' && !Array.isArray(value)

/**
 * Materialize untranslated reference keys in a locale bundle.
 *
 * Vue I18n still keeps its runtime fallback for dynamic misses, but shipped
 * bundles expose one stable structure to tests and tooling. Existing
 * translations always win; missing leaves retain the English source text.
 */
export const completeLocale = (
  reference: LocaleMessages,
  translated: LocaleMessages,
): LocaleMessages => {
  for (const [key, referenceValue] of Object.entries(reference)) {
    const translatedValue = translated[key]
    if (translatedValue === undefined) {
      translated[key] = isMessages(referenceValue)
        ? completeLocale(referenceValue, {})
        : referenceValue
      continue
    }
    if (isMessages(referenceValue) && isMessages(translatedValue)) {
      completeLocale(referenceValue, translatedValue)
    }
  }
  return translated
}
