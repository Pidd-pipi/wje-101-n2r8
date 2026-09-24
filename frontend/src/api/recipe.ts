import request from '@/utils/request'
import type { BrewRecipe, BrewRecipeVersion, RecipeDetail, RecipeListItem } from '@/types/api'
import type { PageData } from '@/types/api'

export function listRecipes(params: { page?: number; page_size?: number; device?: string; keyword?: string }) {
  return request.get<never, PageData<RecipeListItem>>('/recipes', { params })
}
export function getRecipe(id: number | string) { return request.get<never, RecipeDetail>(`/recipes/${id}`) }
export function createRecipe(payload: Partial<BrewRecipe> & { water_temp?: number; grind_size?: string; ratio?: string; steps?: string }) {
  return request.post<never, { recipe: BrewRecipe; version: BrewRecipeVersion }>('/recipes', payload)
}
export function getRecipeVersion(id: number | string, versionNumber: number) {
  return request.get<never, BrewRecipeVersion>(`/recipes/${id}/versions/${versionNumber}`)
}
export function publishRecipeVersion(id: number | string, payload: {
  name?: string
  device?: string
  water_temp: number
  grind_size?: string
  ratio?: string
  steps?: string
}) {
  return request.post<never, { recipe: BrewRecipe; version: BrewRecipeVersion }>(`/recipes/${id}/versions`, payload)
}
export function setActiveRecipeVersion(id: number | string, versionId: number) {
  return request.put<never, { version: BrewRecipeVersion }>(`/recipes/${id}/active-version`, { version_id: versionId })
}
