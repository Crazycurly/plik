<script setup>
import { humanReadableSize, getUploadUrl, formatDate } from '../utils.js'
import { getFileURL } from '../api.js'
import UploadBadges from './UploadBadges.vue'

defineProps({
    upload: { type: Object, required: true },
    tokenLabel: { type: String, default: '' },  // pre-formatted token label
    showUser: { type: Boolean, default: false },
})

const emit = defineEmits(['delete', 'filter-token', 'filter-user'])
</script>

<template>
  <div class="glass-card p-4">
    <div class="flex flex-col sm:flex-row gap-4">
      <!-- Upload meta -->
      <div class="sm:w-1/3 text-sm space-y-1">
        <a :href="getUploadUrl(upload)"
           class="font-mono text-accent-400 hover:text-accent-300 transition-colors">
          {{ upload.id }}
        </a>
        <p class="text-surface-500">uploaded: {{ formatDate(upload.createdAt) }}</p>
        <p class="text-surface-500">expires: {{ upload.expireAt ? formatDate(upload.expireAt) : 'Never' }}</p>
        <UploadBadges :upload="upload" size="sm" class="mt-1" />
        <p v-if="showUser && upload.user" class="text-surface-500">
          user:
          <button @click="emit('filter-user', upload.user)"
                  class="text-accent-400 hover:text-accent-300 transition-colors">
            {{ upload.user }}
          </button>
        </p>
        <p v-if="upload.token" class="text-surface-500">
          token:
          <button @click="emit('filter-token', upload.token)"
                  class="text-accent-400 hover:text-accent-300 transition-colors">
            {{ tokenLabel || upload.token?.substring(0, 8) + '...' }}
          </button>
        </p>
      </div>

      <!-- Files -->
      <div class="flex-1 min-w-0 text-sm space-y-1">
        <div v-for="file in (upload.files || []).filter(f => f.status === 'uploaded')"
             :key="file.id"
             class="flex items-center justify-between gap-2">
          <a :href="getFileURL(upload.id, file.id, file.fileName, upload.stream)"
             class="text-surface-300 hover:text-accent-400 transition-colors truncate">
            {{ file.fileName }}
          </a>
          <span class="text-surface-500 shrink-0">{{ humanReadableSize(file.fileSize) }}</span>
        </div>
        <p v-if="!upload.files || upload.files.length === 0"
           class="text-surface-500 italic">No files</p>
      </div>

      <!-- Actions -->
      <div class="sm:w-20 flex sm:flex-col items-center sm:justify-center gap-2">
        <button @click="emit('delete', upload)"
                class="text-xs text-red-400 hover:text-red-300 border border-red-500/30
                       rounded-lg px-3 py-1.5 hover:bg-red-500/10 transition-colors">
          Remove
        </button>
      </div>
    </div>
  </div>
</template>
