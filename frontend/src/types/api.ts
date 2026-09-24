export interface ApiResponse<T = unknown> { code: number; message: string; data: T }
export interface PageData<T> { list: T[]; total: number; page: number; page_size: number }

export interface BrewRecipe {
  id: number
  user_id: number
  name: string
  device: string
  water_temp: number
  grind_size: string
  ratio: string
  steps: string
  current_version_id: number
  current_version_number?: number
  created_at: string
}

export type RecipeVersionStatus = 'active' | 'deprecated'

export interface RecipeVersion {
  id: number
  recipe_id: number
  version_number: number
  status: RecipeVersionStatus
  name: string
  device: string
  water_temp: number
  grind_size: string
  ratio: string
  steps: string
  created_at: string
  is_current: boolean
}

export interface RecipeDetail extends BrewRecipe {
  versions: RecipeVersion[]
}

// Snapshot frozen on a tasting note at publish time; always reflects the
// exact brew, even if the recipe has newer/deprecated versions since.
export interface RecipeSnapshot {
  recipe_id: number
  version_id: number
  version_number: number
  name: string
  device: string
  water_temp: number
  grind_size: string
  ratio: string
  steps: string
  status: string
  deprecated: boolean
}

export interface Comment {
  id: number
  note_id: number
  user_id: number
  content: string
  created_at: string
}

export interface RecipeStep {
  step_number: number
  description: string
  duration_seconds: number
}
