<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getAvailableThemes, currentTheme, setUserTheme } from '../settings.js'

const open = ref(false)
const pickerRef = ref(null)

const themes = computed(() => getAvailableThemes())
const showPicker = computed(() => themes.value.length > 1)

function toggle() {
    open.value = !open.value
}

async function select(name) {
    await setUserTheme(name)
    open.value = false
}

function onClickOutside(e) {
    if (pickerRef.value && !pickerRef.value.contains(e.target)) {
        open.value = false
    }
}

onMounted(() => document.addEventListener('click', onClickOutside, true))
onUnmounted(() => document.removeEventListener('click', onClickOutside, true))
</script>

<template>
  <div v-if="showPicker" ref="pickerRef" class="relative">
    <!-- Trigger button (desktop: icon only, mobile: icon + text via slot) -->
    <button
        id="theme-picker-toggle"
        class="btn-ghost text-sm"
        :class="{ 'bg-surface-700/50 text-surface-100': open }"
        @click.stop="toggle"
        title="Switch theme">
      <!-- Palette icon -->
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012
                 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2
                 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0
                 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486
                 8.485M7 17h.01" />
      </svg>
      <slot />
    </button>

    <!-- Dropdown -->
    <Transition name="dropdown">
      <div v-if="open"
           class="absolute right-0 top-full mt-2 w-52
                  bg-surface-900 border border-surface-700/50
                  rounded-lg shadow-xl overflow-hidden z-50">
        <div class="py-1 max-h-80 overflow-y-auto">
          <button
              v-for="t in themes"
              :key="t.name"
              :id="'theme-option-' + t.name"
              class="w-full flex items-center gap-3 px-3 py-2 text-sm text-left
                     transition-colors hover:bg-surface-700/30"
              :class="currentTheme === t.name
                ? 'text-accent-400'
                : 'text-surface-300 hover:text-surface-100'"
              @click="select(t.name)">

            <!-- Check mark for active theme -->
            <svg v-if="currentTheme === t.name"
                 class="w-4 h-4 shrink-0 text-accent-400"
                 fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M5 13l4 4L19 7" />
            </svg>
            <span v-else class="w-4 h-4 shrink-0"></span>

            <span class="truncate">{{ t.label }}</span>
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
    transition: opacity 0.15s ease, transform 0.15s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
    opacity: 0;
    transform: translateY(-4px);
}
</style>
