<template>
  <div class="page">
    <h1>配方广场</h1>
    <SearchFilter @search="onSearch" @reset="onReset">
      <template #filters>
        <el-form-item label="器具">
          <el-select v-model="device" clearable placeholder="全部器具" style="width: 160px" @change="load">
            <el-option v-for="d in DEVICES" :key="d" :label="d" :value="d" />
          </el-select>
        </el-form-item>
      </template>
    </SearchFilter>
    <el-row :gutter="16">
      <el-col v-for="r in recipes" :key="r.id" :xs="24" :sm="12" :md="8">
        <el-card class="recipe-card" shadow="hover">
          <h3>
            {{ r.name }}
            <el-tag size="small">{{ r.device }}</el-tag>
            <el-tag size="small" type="success" effect="plain">下次用 v{{ r.current_version }}</el-tag>
          </h3>
          <div v-if="r.active_version" class="meta">
            {{ r.active_version.water_temp }}°C · {{ r.active_version.grind_size }} · 粉水比 {{ r.active_version.ratio }}
          </div>
          <ol v-if="r.active_version">
            <li v-for="st in stepsOf(r.active_version.steps)" :key="st.step_number">
              第{{ st.step_number }}步：{{ st.description }}（{{ st.duration_seconds }}s）
            </li>
          </ol>
          <div class="card-actions">
            <el-button link type="primary" size="small" @click="openHistory(r.id)">版本历史</el-button>
            <el-button v-if="isOwner(r)" link type="warning" size="small" @click="openPublish(r)">发布新版本</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <EmptyState v-if="!recipes.length" description="暂无配方" />
    <el-button v-if="isLoggedIn" type="primary" style="margin-top: 16px" @click="openCreate">分享我的配方</el-button>

    <!-- 新建配方（同时生成 v1） -->
    <el-dialog v-model="createVisible" title="分享冲煮配方" width="520px">
      <el-form label-width="80px">
        <el-form-item label="名称"><el-input v-model="addForm.name" /></el-form-item>
        <el-form-item label="器具">
          <el-select v-model="addForm.device" style="width: 200px">
            <el-option v-for="d in DEVICES" :key="d" :label="d" :value="d" />
          </el-select>
        </el-form-item>
        <el-form-item label="水温"><el-input-number v-model="addForm.water_temp" :min="80" :max="100" /></el-form-item>
        <el-form-item label="研磨度"><el-input v-model="addForm.grind_size" /></el-form-item>
        <el-form-item label="粉水比"><el-input v-model="addForm.ratio" placeholder="1:15" /></el-form-item>
        <el-form-item label="步骤(JSON)"><el-input v-model="addForm.steps" type="textarea" :rows="4" placeholder='[{"step_number":1,"description":"闷蒸","duration_seconds":30}]' /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" @click="addRecipe">发布</el-button>
      </template>
    </el-dialog>

    <!-- 版本历史 -->
    <el-dialog v-model="historyVisible" :title="`版本历史：${detail?.recipe.name ?? ''}`" width="640px">
      <template v-if="detail">
        <el-alert type="info" :closable="false" class="hist-tip"
          title="只有作者能修改配方或指定下次使用版本。已作废版本保留全部数据，供旧笔记核对。" />
        <el-timeline>
          <el-timeline-item v-for="v in detail.versions" :key="v.id" :timestamp="formatDateTime(v.created_at)" placement="top">
            <el-card shadow="never" class="ver-card">
              <div class="ver-head">
                <strong>v{{ v.version_number }}</strong>
                <el-tag v-if="v.status === 'active'" size="small" type="success">下次使用</el-tag>
                <el-tag v-else size="small" type="info" effect="plain">已作废</el-tag>
                <el-button v-if="isOwner(detail.recipe) && v.status !== 'active'" link type="primary" size="small"
                  @click="useVersion(v.id)">指定下次使用此版</el-button>
              </div>
              <div class="meta">{{ v.water_temp }}°C · {{ v.grind_size }} · 粉水比 {{ v.ratio }}</div>
              <ol class="ver-steps">
                <li v-for="st in stepsOf(v.steps)" :key="st.step_number">
                  第{{ st.step_number }}步：{{ st.description }}（{{ st.duration_seconds }}s）
                </li>
              </ol>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </template>
    </el-dialog>

    <!-- 发布新版本 -->
    <el-dialog v-model="publishVisible" title="发布新版本" width="520px">
      <el-form label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="publishForm.name" :placeholder="detail?.recipe.name" />
        </el-form-item>
        <el-form-item label="器具">
          <el-select v-model="publishForm.device" style="width: 200px" :placeholder="detail?.recipe.device">
            <el-option v-for="d in DEVICES" :key="d" :label="d" :value="d" />
          </el-select>
        </el-form-item>
        <el-form-item label="水温"><el-input-number v-model="publishForm.water_temp" :min="80" :max="100" /></el-form-item>
        <el-form-item label="研磨度"><el-input v-model="publishForm.grind_size" /></el-form-item>
        <el-form-item label="粉水比"><el-input v-model="publishForm.ratio" placeholder="1:15" /></el-form-item>
        <el-form-item label="步骤(JSON)"><el-input v-model="publishForm.steps" type="textarea" :rows="4" /></el-form-item>
      </el-form>
      <div class="ver-hint">保存后自动生成连续的下一版本号并成为「下次使用」版本，历史版本不会被改动。</div>
      <template #footer>
        <el-button @click="publishVisible = false">取消</el-button>
        <el-button type="primary" :loading="publishing" @click="submitVersion">发布新版本</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import SearchFilter from '@/components/common/SearchFilter.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import {
  listRecipes, createRecipe, getRecipe, publishRecipeVersion, setActiveRecipeVersion,
} from '@/api/recipe'
import { useAuth } from '@/hooks/useAuth'
import type { RecipeListItem, RecipeDetail, RecipeStep } from '@/types/api'
import { formatDateTime } from '@/utils/dateFormat'

const { isLoggedIn, user } = useAuth()
const recipes = ref<RecipeListItem[]>([])
const device = ref('')
const keyword = ref('')

const createVisible = ref(false)
const addForm = reactive({ name: '', device: '手冲壶', water_temp: 92, grind_size: '中细', ratio: '1:15', steps: '[]' })

const historyVisible = ref(false)
const detail = ref<RecipeDetail | null>(null)

const publishVisible = ref(false)
const publishing = ref(false)
const publishForm = reactive({ name: '', device: '', water_temp: 92, grind_size: '中细', ratio: '1:15', steps: '[]' })

const DEVICES = ['手冲壶', '法压壶', '意式机', '爱乐压', '冷萃壶']

onMounted(() => load())

function isOwner(r: { user_id: number }): boolean {
  return !!user.value && r.user_id === user.value.id
}

async function load() {
  const res = await listRecipes({ page: 1, page_size: 20, device: device.value, keyword: keyword.value })
  recipes.value = res.list
}
function onSearch(kw: string) {
  keyword.value = kw
  load()
}
function onReset() {
  device.value = ''
  keyword.value = ''
  load()
}
function stepsOf(raw: string): RecipeStep[] {
  try {
    return JSON.parse(raw || '[]')
  } catch {
    return []
  }
}

function openCreate() {
  Object.assign(addForm, { name: '', device: '手冲壶', water_temp: 92, grind_size: '中细', ratio: '1:15', steps: '[]' })
  createVisible.value = true
}
async function addRecipe() {
  if (!addForm.name) {
    ElMessage.warning('请填写配方名称')
    return
  }
  await createRecipe({ ...addForm })
  ElMessage.success('配方已分享（v1）')
  createVisible.value = false
  await load()
}

async function openHistory(id: number) {
  detail.value = await getRecipe(id)
  historyVisible.value = true
}

function openPublish(r: RecipeListItem) {
  const active = recipes.value.find((x) => x.id === r.id)?.active_version
  Object.assign(publishForm, {
    name: '',
    device: '',
    water_temp: active?.water_temp ?? 92,
    grind_size: active?.grind_size ?? '中细',
    ratio: active?.ratio ?? '1:15',
    steps: active?.steps ?? '[]',
  })
  // 确保历史弹窗数据存在（发布后刷新同一详情）
  if (!detail.value || detail.value.recipe.id !== r.id) {
    getRecipe(r.id).then((d) => (detail.value = d))
  }
  publishVisible.value = true
}

async function submitVersion() {
  if (!detail.value) return
  if (!publishForm.steps.trim()) publishForm.steps = '[]'
  publishing.value = true
  try {
    await publishRecipeVersion(detail.value.recipe.id, { ...publishForm })
    ElMessage.success('新版本已发布')
    publishVisible.value = false
    detail.value = await getRecipe(detail.value.recipe.id)
    await load()
  } finally {
    publishing.value = false
  }
}

async function useVersion(versionId: number) {
  if (!detail.value) return
  try {
    await ElMessageBox.confirm('指定该版本为下一次冲煮使用？其余版本将标记为已作废，但数据保留。', '确认', { type: 'warning' })
  } catch {
    return
  }
  await setActiveRecipeVersion(detail.value.recipe.id, versionId)
  ElMessage.success('已更新下次使用版本')
  detail.value = await getRecipe(detail.value.recipe.id)
  await load()
}
</script>

<style scoped>
.page { max-width: 1200px; margin: 0 auto; }
.recipe-card { margin-bottom: 16px; }
.meta { color: #999; font-size: 12px; margin: 6px 0; }
.card-actions { margin-top: 8px; display: flex; gap: 12px; }
.hist-tip { margin-bottom: 12px; }
.ver-card { background: #fafafa; }
.ver-head { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.ver-steps { margin: 6px 0 0; padding-left: 20px; font-size: 13px; color: #555; }
.ver-hint { color: #999; font-size: 12px; margin-top: 4px; }
</style>
