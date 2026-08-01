export type RecommendationPath = string | Array<string | number>
export type RecommendationMode = 'create' | 'edit' | 'both'

export interface RecommendationContext<TModel extends object> {
  mode: RecommendationMode
  model: TModel
}

export interface RecommendationSpec<TModel extends object> {
  id: string
  labelKey: string
  descriptionKey?: string
  path: RecommendationPath
  value: unknown | ((context: RecommendationContext<TModel>) => unknown)
  mode?: RecommendationMode
  onlyIfEmpty?: boolean
}

export interface ResolvedRecommendation<TModel extends object> extends RecommendationSpec<TModel> {
  currentValue: unknown
  recommendedValue: unknown
  applicable: boolean
}

export const recommendationPath = (path: RecommendationPath): Array<string | number> => {
  if (Array.isArray(path)) return path
  if (path.length === 0) return []

  return path.split('.').filter(Boolean).map((part) => {
    const index = Number(part)
    return Number.isInteger(index) && String(index) === part ? index : part
  })
}

export const recommendationValue = (target: unknown, path: RecommendationPath): unknown =>
  recommendationPath(path).reduce<unknown>((current, part) => {
    if (current === null || typeof current !== 'object') return undefined
    return (current as Record<string | number, unknown>)[part]
  }, target)

export const isEmptyRecommendationValue = (value: unknown): boolean => {
  if (value == null || value === '') return true
  if (typeof value === 'number') return value === 0
  if (Array.isArray(value)) return value.length === 0
  if (typeof value === 'object') return Object.keys(value).length === 0
  return false
}

const modeMatches = <TModel extends object>(
  spec: RecommendationSpec<TModel>,
  context: RecommendationContext<TModel>,
): boolean => spec.mode == null || spec.mode === 'both' || spec.mode === context.mode

const resolvedValue = <TModel extends object>(
  spec: RecommendationSpec<TModel>,
  context: RecommendationContext<TModel>,
): unknown => typeof spec.value === 'function' ? spec.value(context) : spec.value

export const resolveRecommendations = <TModel extends object>(
  specs: RecommendationSpec<TModel>[],
  context: RecommendationContext<TModel>,
): ResolvedRecommendation<TModel>[] => specs
  .filter((spec) => modeMatches(spec, context))
  .map((spec) => {
    const currentValue = recommendationValue(context.model, spec.path)
    return {
      ...spec,
      currentValue,
      recommendedValue: resolvedValue(spec, context),
      applicable: !(spec.onlyIfEmpty ?? true) || isEmptyRecommendationValue(currentValue),
    }
  })

export const applyRecommendation = <TModel extends object>(
  spec: RecommendationSpec<TModel>,
  context: RecommendationContext<TModel>,
): boolean => {
  if (!modeMatches(spec, context)) return false
  const parts = recommendationPath(spec.path)
  const currentValue = recommendationValue(context.model, parts)
  if ((spec.onlyIfEmpty ?? true) && !isEmptyRecommendationValue(currentValue)) return false
  if (parts.length === 0) return false

  let target = context.model as Record<string | number, unknown>
  for (let index = 0; index < parts.length - 1; index += 1) {
    const part = parts[index]
    const nextPart = parts[index + 1]
    const nextValue = target[part]
    if (nextValue === null || typeof nextValue !== 'object') {
      target[part] = typeof nextPart === 'number' ? [] : {}
    }
    target = target[part] as Record<string | number, unknown>
  }
  target[parts[parts.length - 1]] = resolvedValue(spec, context)
  return true
}
