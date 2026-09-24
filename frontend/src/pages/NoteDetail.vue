<template>
  <div class="page" v-if="note">
    <el-page-header @back="$router.back()" content="品鉴详情" />
    <el-row :gutter="16">
      <el-col :xs="24" :md="14">
        <el-card>
          <el-image v-if="note.image_url" :src="note.image_url" fit="cover" class="cover" />
          <h1>{{ note.coffee_name }}</h1>
          <div class="meta">{{ note.origin || '-' }} · {{ RoastLevelMap[note.roast_level] }} · {{ note.brew_method || '-' }}</div>
          <ScoreStars :model-value="note.overall_score" />
          <FlavorTags :tags="note.flavor_tags" />
          <el-descriptions :column="2" border class="scores">
            <el-descriptions-item label="香气">{{ note.aroma_score }}</el-descriptions-item>
            <el-descriptions-item label="酸质">{{ note.acidity_score }}</el-descriptions-item>
            <el-descriptions-item label="醇厚">{{ note.body_score }}</el-descriptions-item>
            <el-descriptions-item label="综合">{{ note.overall_score }}</el-descriptions-item>
          </el-descriptions>
          <p class="notes">{{ note.notes_text }}</p>
          <div class="actions">
            <el-button :type="liked ? 'warning' : 'default'" :loading="liking" @click="toggleLike">
              👍 {{ likeCount }}
            </el-button>
            <el-button v-if="isOwner" type="danger" plain @click="remove">删除</el-button>
          </div>
        </el-card>
        <!-- 配方数据为发笔记时固化的快照，配方后续改版不会影响这里 -->
        <el-card v-if="note.brew_recipe_id" class="block">
          <template #header>
            关联配方：{{ note.recipe_name_snapshot || '（未命名）' }}
            <el-tag size="small" type="info" effect="plain">v{{ note.recipe_version_snapshot }}</el-tag>
            <el-button v-if="canViewCurrent" link type="primary" size="small" @click="openCurrent">查看配方当前版本</el-button>
          </template>
          <p class="snap-hint">以下为本次冲煮时采用的数据</p>
          <p>水温 {{ note.recipe_temp_snapshot }}°C</p>
          <ol>
            <li v-for="s in steps" :key="s.step_number">
              第{{ s.step_number }}步：{{ s.description }}（{{ s.duration_seconds }}s）
            </li>
          </ol>
        </el-card>
        <el-dialog v-model="currentVisible" :title="`配方最新状态：${current?.recipe.name ?? ''}`" width="560px">
          <template v-if="current">
            <p v-if="current.active_version">
              当前启用版本：<el-tag size="small">v{{ current.active_version.version_number }}</el-tag>
              · {{ current.active_version.water_temp }}°C · {{ current.active_version.grind_size }} · 粉水比 {{ current.active_version.ratio }}
            </p>
            <el-alert v-if="note.recipe_version_snapshot !== current.recipe.current_version" type="warning" :closable="false"
              :title="`本笔记冲煮时使用的是 v${note.recipe_version_snapshot}，配方现已有 v${current.recipe.current_version}，上方记录保持不变。`" />
          </template>
        </el-dialog>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card>
          <template #header>评论（{{ comments.length }}）</template>
          <div v-for="c in comments" :key="c.id" class="comment">
            <div class="c-head">用户 #{{ c.user_id }} · {{ formatDateTime(c.created_at) }}</div>
            <div>{{ c.content }}</div>
          </div>
          <el-empty v-if="!comments.length" description="暂无评论" />
          <div class="reply">
            <el-input v-model="reply" type="textarea" :rows="3" placeholder="写下你的评论…" />
            <el-button type="primary" :loading="replying" @click="submitReply">发表评论</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import ScoreStars from '@/components/common/ScoreStars.vue'
import FlavorTags from '@/components/common/FlavorTags.vue'
import { getNote, listComments, createComment, likeNote, unlikeNote, deleteNote } from '@/api/note'
import { getRecipe } from '@/api/recipe'
import { useAuth } from '@/hooks/useAuth'
import { RoastLevelMap, type TastingNote } from '@/constants/note'
import type { Comment, RecipeStep, RecipeDetail } from '@/types/api'
import { formatDateTime } from '@/utils/dateFormat'

const route = useRoute()
const router = useRouter()
const { isLoggedIn, user } = useAuth()
const note = ref<TastingNote | null>(null)
const likeCount = ref(0)
const liked = ref(false)
const liking = ref(false)
const comments = ref<Comment[]>([])
const reply = ref('')
const replying = ref(false)
const currentVisible = ref(false)
const current = ref<RecipeDetail | null>(null)

const steps = computed<RecipeStep[]>(() => {
  try {
    return JSON.parse(note.value?.recipe_steps_snapshot || '[]')
  } catch {
    return []
  }
})
const canViewCurrent = computed(() => !!note.value?.brew_recipe_id)
const isOwner = computed(() => !!user.value && note.value?.user_id === user.value.id)

onMounted(async () => {
  const id = route.params.id as string
  const res = await getNote(id)
  note.value = res.note
  likeCount.value = res.like_count
  comments.value = await listComments(res.note.id)
})

async function openCurrent() {
  if (!note.value?.brew_recipe_id) return
  current.value = await getRecipe(note.value.brew_recipe_id)
  currentVisible.value = true
}

async function toggleLike() {
  if (!isLoggedIn.value) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  liking.value = true
  try {
    if (liked.value) {
      await unlikeNote(note.value!.id)
      liked.value = false
      likeCount.value = Math.max(0, likeCount.value - 1)
    } else {
      await likeNote(note.value!.id)
      liked.value = true
      likeCount.value += 1
    }
  } finally {
    liking.value = false
  }
}

async function submitReply() {
  if (!isLoggedIn.value) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  if (!reply.value.trim()) return
  replying.value = true
  try {
    await createComment(note.value!.id, reply.value)
    comments.value = await listComments(note.value!.id)
    reply.value = ''
  } finally {
    replying.value = false
  }
}

async function remove() {
  await deleteNote(note.value!.id)
  ElMessage.success('笔记已删除')
  router.push('/')
}
</script>

<style scoped>
.page { max-width: 1000px; margin: 0 auto; }
.cover { width: 100%; max-height: 360px; border-radius: 8px; }
.meta { color: #999; margin: 8px 0; }
.scores { margin-top: 12px; }
.notes { line-height: 1.8; margin-top: 12px; }
.actions { margin-top: 16px; display: flex; gap: 12px; }
.block { margin-top: 16px; }
.snap-hint { color: #e6a23c; font-size: 12px; margin: 0 0 8px; }
.comment { border-bottom: 1px solid #f0f0f0; padding: 10px 0; }
.c-head { color: #999; font-size: 12px; }
.reply { margin-top: 12px; display: flex; flex-direction: column; gap: 10px; }
</style>
