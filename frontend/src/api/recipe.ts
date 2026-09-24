import request from '@/utils/request'
import type { BrewRecipe, RecipeDetail, RecipeVersion, RecipeVersionStatus } from '@/types/api'
import type { PageData } from '@/types/api'

export function listRecipes(params: { page?: number; page_size?: number; device?: string; keyword?: string }) {
  return request.get<never, PageData<BrewRecipe>>('/recipes', { params })
}
export function getRecipe(id: number | string) { return request.get<never, RecipeDetail>(`/recipes/${id}`) }
export function createRecipe(payload: Partial<BrewRecipe>) { return request.post<never, BrewRecipe>('/recipes', payload) }

// Publish a new immutable version; it becomes the current version.
export function createRecipeVersion(recipeId: number, payload: {
  name: string
  device?: string
  water_temp: number
  grind_size?: string
  ratio?: string
  steps?: string
}) {
  return request.post<never, RecipeVersion>(`/recipes/${recipeId}/versions`, payload)
}

// Select which version is used the next time someone brews with this recipe.
export function setCurrentRecipeVersion(recipeId: number, versionNumber: number) {
  return request.put<never, { current_version_number: number }>(`/recipes/${recipeId}/current-version`, {
    version_number: versionNumber,
  })
}

// Mark a version deprecated or restore it to active. Deprecated versions stay
// readable for old notes but cannot be chosen as current.
export function setRecipeVersionStatus(recipeId: number, versionNumber: number, status: RecipeVersionStatus) {
  return request.put<never, { version_number: number; status: RecipeVersionStatus }>(
    `/recipes/${recipeId}/versions/${versionNumber}/status`,
    { status },
  )
}
