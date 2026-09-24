<template>
  <div class="page">
    <h1>创建品鉴笔记</h1>
    <el-form label-width="90px" class="form">
      <el-form-item label="豆种">
        <el-select v-model="beanId" placeholder="选择豆种自动填充" filterable clearable style="width: 320px" @change="onBeanChange">
          <el-option v-for="b in beans" :key="b.id" :label="`${b.name}（${b.origin}）`" :value="b.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="咖啡名称"><el-input v-model="form.coffee_name" placeholder="如：埃塞俄比亚耶加雪菲" /></el-form-item>
      <el-form-item label="产地"><el-input v-model="form.origin" /></el-form-item>
      <el-form-item label="烘焙度">
        <el-radio-group v-model="form.roast_level">
          <el-radio-button v-for="(label, value) in RoastLevelMap" :key="value" :value="value">{{ label }}</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="风味标签"><FlavorTags :tags="form.flavor_tags" /></el-form-item>
      <el-form-item label="风味"><el-select v-model="tagInput" filterable allow-create default-first-option multiple style="width: 400px" placeholder="输入风味并回车" /></el-form-item>
      <el-form-item label="香气分"><el-rate v-model="form.aroma_score" :max="10" show-score /></el-form-item>
      <el-form-item label="酸质分"><el-rate v-model="form.acidity_score" :max="10" show-score /></el-form-item>
      <el-form-item label="醇厚度"><el-rate v-model="form.body_score" :max="10" show-score /></el-form-item>
      <el-form-item label="综合分"><el-rate v-model="form.overall_score" :max="10" show-score /></el-form-item>
      <el-form-item label="冲煮方式"><el-input v-model="form.brew_method" placeholder="如：手冲" /></el-form-item>
      <el-form-item label="关联配方">
        <el-select v-model="form.brew_recipe_id" clearable placeholder="选择冲煮配方" style="width: 320px" @change="onRecipeChange">
          <el-option v-for="r in recipes" :key="r.id" :label="`${r.name}（${r.device}）`" :value="r.id" />
        </el-select>
      </el-form-item>
      <el-form-item v-if="recipeDetail" label="采用版本">
        <el-radio-group v-model="form.brew_recipe_version_id" @change="onVersionChange">
          <el-radio v-for="v in recipeDetail.versions" :key="v.id" :value="v.id" :disabled="false">
            v{{ v.version_number }} · {{ v.water_temp }}°C
            <el-tag v-if="v.status === 'deprecated'" size="small" type="info" effect="plain">已作废</el-tag>
            <el-tag v-else size="small" type="success">下次使用</el-tag>
          </el-radio>
        </el-radio-group>
        <div class="ver-hint">发布笔记时会固化所选版本的名称、水温和步骤；默认选中作者指定的下次使用版本，也可改选旧版本核对冲煮。</div>
      </el-form-item>
      <el-form-item label="品鉴笔记"><el-input v-model="form.notes_text" type="textarea" :rows="4" /></el-form-item>
      <el-form-item label="配图"><ImageUploader v-model="form.image_url" /></el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="submitting" @click="submit">发布笔记</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import FlavorTags from '@/components/common/FlavorTags.vue'
import ImageUploader from '@/components/common/ImageUploader.vue'
import { createNote } from '@/api/note'
import { listBeans } from '@/api/bean'
import { listRecipes, getRecipe } from '@/api/recipe'
import { RoastLevelMap, parseTags, type RoastLevel } from '@/constants/note'
import type { CoffeeBean } from '@/constants/bean'
import type { BrewRecipe, RecipeDetail } from '@/types/api'

const router = useRouter()
const beans = ref<CoffeeBean[]>([])
const recipes = ref<BrewRecipe[]>([])
const beanId = ref<number>()
const tagInput = ref<string[]>([])
const submitting = ref(false)
const recipeDetail = ref<RecipeDetail | null>(null)
const form = reactive({
  coffee_name: '', origin: '', roast_level: 'light' as string, flavor_tags: '[]',
  aroma_score: 0, acidity_score: 0, body_score: 0, overall_score: 0,
  brew_method: '', brew_recipe_id: 0, brew_recipe_version_id: 0, notes_text: '', image_url: '',
})

watch(tagInput, (v) => {
  form.flavor_tags = JSON.stringify(v)
})

onMounted(async () => {
  beans.value = (await listBeans({ page_size: 100 })).list
  recipes.value = (await listRecipes({ page_size: 100 })).list
})

function onBeanChange(id: number | undefined) {
  const b = beans.value.find((x) => x.id === id)
  if (!b) return
  form.coffee_name = b.name
  form.origin = b.origin
  tagInput.value = parseTags(b.flavor_tags)
  form.flavor_tags = b.flavor_tags
}

async function onRecipeChange(id: number | undefined) {
  form.brew_recipe_version_id = 0
  recipeDetail.value = null
  if (!id) return
  recipeDetail.value = await getRecipe(id)
  if (recipeDetail.value.active_version) {
    form.brew_recipe_version_id = recipeDetail.value.active_version.id
  }
}

function onVersionChange() {
  // 版本仅决定快照内容；服务端按所选版本固化
}

async function submit() {
  if (!form.coffee_name || !form.roast_level) {
    ElMessage.warning('请填写咖啡名称与烘焙度')
    return
  }
  submitting.value = true
  try {
    const note = await createNote({
      ...form,
      roast_level: form.roast_level as RoastLevel,
      brew_recipe_id: form.brew_recipe_id || 0,
      brew_recipe_version_id: form.brew_recipe_id ? form.brew_recipe_version_id || 0 : 0,
    })
    ElMessage.success('品鉴笔记已发布')
    router.push(`/note/${note.id}`)
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.page { max-width: 720px; margin: 0 auto; }
.form { margin-top: 16px; }
.ver-hint { color: #999; font-size: 12px; margin-top: 6px; }
</style>
