<template>
  <Transition name="fade">
    <div v-if="show" class="update-banner">
      <div class="update-banner-content">
        <div class="update-banner-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
            <path d="M3 3v5h5"/>
            <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/>
            <path d="M16 16h5v5"/>
          </svg>
        </div>
        <h2 class="update-banner-title">{{ title }}</h2>
        <p class="update-banner-message">{{ message }}</p>
        <div v-if="isUpdating" class="update-banner-spinner">
          <div class="spinner"></div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  },
  isUpdating: {
    type: Boolean,
    default: false
  }
})

const title = computed(() =>
  props.isUpdating ? 'Aktualizacja...' : 'Nowa wersja aplikacji'
)

const message = computed(() =>
  props.isUpdating ? 'Strona zaraz się odświeży' : 'Wykryto nową wersję. Aktualizacja...'
)
</script>

<style scoped>
.update-banner {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.95);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.update-banner-content {
  text-align: center;
  padding: 3rem;
  max-width: 500px;
}

.update-banner-icon {
  color: #3b82f6;
  margin-bottom: 1.5rem;
  animation: rotate 2s linear infinite;
}

.update-banner-icon svg {
  display: inline-block;
}

.update-banner-title {
  font-size: 2rem;
  font-weight: 700;
  color: #fff;
  margin-bottom: 1rem;
}

.update-banner-message {
  font-size: 1.125rem;
  color: #9ca3af;
  margin-bottom: 2rem;
}

.update-banner-spinner {
  display: flex;
  justify-content: center;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid rgba(59, 130, 246, 0.2);
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
