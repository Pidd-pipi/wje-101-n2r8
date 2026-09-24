<template>
  <div class="page" v-loading="loading">
    <el-page-header @back="$router.push('/recipes')" content="配方详情" />
    <template v-if="detail">
      <el-card class="head-card">
        <div class="head">
          <div>
            <h1>
              {{ detail.name }}
              <el-tag size="small">{{ detail.device }}</el-tag>
              <el-tag size="small" type="success" v-if="currentVersion">
                当前使用 v{{ currentVersion.version_number }}
              </el-tag>
              <el-tag v-if="isOwner" size="small" type="warning">我的配方</el-tag>
            </h1>
            <div class="meta">
              {{ detail.water_temp }}°C · {{ detail.grind_size || '-' }} · 粉水比 {{ detail.ratio || '-' }}
            </div>
          </div>
          <el-button v-if="isOwner" type="primary" @click="openNewVersion">发布新版本</el-button>
        </div>
        <ol class="steps">
          <li v-for="s in currentSteps" :key="s.step_number">
            第{{ s.step_number }}步：{{ s.description }}（{{ s.duration_seconds }}s）
          </li>
        </ol>
      </el-card>

      <h2 class="history-title">
        版本历史
        <span class="sub">共 {{ detail.versions.length }} 个版本，旧笔记始终引用其发布时的版本</span>
      </h2>
      <el-timeline>
        <el-timeline-item v-for="v in detail.versions" :key="v.id" :timestamp="formatDateTime(v.created_at)" placement="top">
          <el-card :class="['version-card', { deprecated: v.status === 'deprecated' }]">
            <div class="v-head">
              <span class="v-title">v{{ v.version_number }} · {{ v.name }}</span>
              <el-tag v-if="v.is_current" size="small" type="success">当前使用</el-tag>
              <el-tag v-if="v.status === 'deprecated'" size="small" type="warning">已作废</el-tag>
              <el-tag v-else-if="!v.is_current" size="small" type="info">历史版本</el-tag>
            </div>
            <div class="meta">{{ v.device || '-' }} · {{ v.water_temp }}°C · {{ v.grind_size || '-' }} · 粉水比 {{ v.ratio || '-' }}</div>
            <ol class="steps compact">
              <li v-for="s in stepsOf(v.steps)" :key="s.step_number">
                第{{ s.step_number }}步：{{ s.description }}（{{ s.duration_seconds }}s）
              </li>
            </ol>
            <div v-if="isOwner" class="v-actions">
              <el-button
                size="small"
                :type="v.is_current ? 'success' : 'default'"
                :disabled="v.is_current || v.status === 'deprecated'"
                @click="useVersion(v)"
              >
                {{ v.is_current ? '正在使用' : '指定下一次使用此版本' }}
              </el-button>
              <el-button
                v-if="v.status === 'active'"
                size="small"
                type="warning"
                plain
                :disabled="v.is_current"
                @click="changeStatus(v, 'deprecated')"
              >
                作废
              </el-button>
              <el-button v-else size="small" type="primary" plain @click="changeStatus(v, 'active')">
                恢复为可用
              </el-button>
            </div>
            <div v-if="v.status === 'deprecated'" class="v-note">
              已作废版本不可指定为当前版本，但仍保留供旧品鉴笔记核对。
            </div>
          </el-card>
        </el-timeline-item>
      </el-timeline>

      <el-dialog v-model="showNew" title="发布配方新版本" width="520px">
        <el-form label-width="80px">
          <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="器具"><el-input v-model="form.device" /></el-form-item>
          <el-form-item label="水温"><el-input-number v-model="form.water_temp" :min="80" :max="100" /></el-form-item>
          <el-form-item label="研磨度"><el-input v-model="form.grind_size" /></el-form-item>
          <el-form-item label="粉水比"><el-input v-model="form.ratio" placeholder="1:15" /></el-form-item>
          <el-form-item label="步骤(JSON)">
            <el-input v-model="form.steps" type="textarea" :rows="4" placeholder='[{"step_number":1,"description":"闷蒸","duration_seconds":30}]' />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showNew = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submitVersion">发布新版本</el-button>
        </template>
      </el-dialog>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute } from 'vue-router'
import {
  getRecipe,
  createRecipeVersion,
  setCurrentRecipeVersion,
  setRecipeVersionStatus,
} from '@/api/recipe'
import { useAuth } from '@/hooks/useAuth'
import { formatDateTime } from '@/utils/dateFormat'
import type { RecipeDetail, RecipeStep, RecipeVersion, RecipeVersionStatus } from '@/types/api'

const route = useRoute()
const { user } = useAuth()
const loading = ref(true)
const detail = ref<RecipeDetail | null>(null)
const showNew = ref(false)
const submitting = ref(false)
const form = reactive({ name: '', device: '手冲壶', water_temp: 92, grind_size: '中细', ratio: '1:15', steps: '[]' })

const currentVersion = computed(() => detail.value?.versions.find((v) => v.is_current) ?? null)
const currentSteps = computed<RecipeStep[]>(() => stepsOf(currentVersion.value?.steps || detail.value?.steps || '[]'))
const isOwner = computed(() => !!user.value && !!detail.value && detail.value.user_id === user.value.id)

onMounted(() => load())

async function load() {
  loading.value = true
  try {
    detail.value = await getRecipe(route.params.id as string)
    if (currentVersion.value) {
      form.name = currentVersion.value.name
      form.device = currentVersion.value.device
      form.water_temp = currentVersion.value.water_temp
      form.grind_size = currentVersion.value.grind_size
      form.ratio = currentVersion.value.ratio
      form.steps = currentVersion.value.steps
    }
  } finally {
    loading.value = false
  }
}

function stepsOf(raw: string): RecipeStep[] {
  try {
    return JSON.parse(raw || '[]')
  } catch {
    return []
  }
}

function openNewVersion() {
  const cur = currentVersion.value
  if (cur) {
    form.name = cur.name
    form.device = cur.device
    form.water_temp = cur.water_temp
    form.grind_size = cur.grind_size
    form.ratio = cur.ratio
    form.steps = cur.steps
  }
  showNew.value = true
}

async function submitVersion() {
  if (!form.name) {
    ElMessage.warning('请填写配方名称')
    return
  }
  if (!detail.value) return
  submitting.value = true
  try {
    await createRecipeVersion(detail.value.id, { ...form })
    ElMessage.success('新版本已发布，并已设为当前版本')
    showNew.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

async function useVersion(v: RecipeVersion) {
  if (!detail.value) return
  try {
    await ElMessageBox.confirm(`确定下一次冲煮使用 v${v.version_number} 吗？`, '指定当前版本', { type: 'info' })
  } catch {
    return
  }
  await setCurrentRecipeVersion(detail.value.id, v.version_number)
  ElMessage.success(`已指定 v${v.version_number} 为下一次使用的版本`)
  await load()
}

async function changeStatus(v: RecipeVersion, status: RecipeVersionStatus) {
  if (!detail.value) return
  const action = status === 'deprecated' ? '作废' : '恢复'
  try {
    await ElMessageBox.confirm(`确定${action} v${v.version_number} 吗？`, `${action}版本`, {
      type: status === 'deprecated' ? 'warning' : 'info',
    })
  } catch {
    return
  }
  await setRecipeVersionStatus(detail.value.id, v.version_number, status)
  ElMessage.success(`v${v.version_number} 已${status === 'deprecated' ? '作废' : '恢复为可用'}`)
  await load()
}
</script>
<style scoped>
.page { max-width: 900px; margin: 0 auto; }
.head-card { margin-top: 16px; }
.head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; }
.head h1 { margin: 0 0 8px; display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.meta { color: #999; font-size: 13px; }
.steps { line-height: 1.9; margin: 12px 0 0; }
.steps.compact { margin-top: 6px; font-size: 13px; }
.history-title { margin: 24px 0 8px; font-size: 18px; }
.history-title .sub { font-size: 12px; color: #999; font-weight: normal; margin-left: 8px; }
.version-card.deprecated { background: #fdfaf3; }
.v-head { display: flex; align-items: center; gap: 8px; }
.v-title { font-weight: 600; }
.v-actions { margin-top: 10px; display: flex; gap: 8px; }
.v-note { margin-top: 8px; color: #b88230; font-size: 12px; }
</style>
