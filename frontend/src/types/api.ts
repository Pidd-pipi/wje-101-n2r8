export interface ApiResponse<T = unknown> { code: number; message: string; data: T }
export interface PageData<T> { list: T[]; total: number; page: number; page_size: number }

// BrewRecipe is the recipe identity; brewing parameters live in its versions.
export interface BrewRecipe {
  id: number
  user_id: number
  name: string
  device: string
  current_version: number
  active_version_id: number
  created_at: string
  updated_at: string
}

export type RecipeVersionStatus = 'active' | 'deprecated'

export interface BrewRecipeVersion {
  id: number
  recipe_id: number
  version_number: number
  water_temp: number
  grind_size: string
  ratio: string
  steps: string
  status: RecipeVersionStatus
  created_at: string
}

export interface RecipeDetail {
  recipe: BrewRecipe
  active_version: BrewRecipeVersion | null
  versions: BrewRecipeVersion[]
}

// List items embed the recipe fields and its active version parameters.
export type RecipeListItem = BrewRecipe & { active_version: BrewRecipeVersion | null }

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
