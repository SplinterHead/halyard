<template>
  <v-app id="halyard" class="bg-transparent">
    <v-navigation-drawer 
      v-if="!isAuthPage"
      v-model="drawer" 
      :rail="rail"
      permanent
      app
      class="glass-panel border-e-0"
      elevation="0"
      @click="rail = false"
    >
      <v-list-item
        class="py-4 mt-2"
        link
        @click.stop="rail = !rail"
      >
        <template v-slot:prepend>
          <v-avatar size="32" class="ms-n1" rounded="0">
            <v-img src="/logo.png" contain />
          </v-avatar>
        </template>
        <v-list-item-title class="text-h6 font-weight-bold text-gradient ms-2" v-if="!rail">
          Halyard
        </v-list-item-title>
      </v-list-item>

      <v-divider class="my-2 border-opacity-25" color="white"></v-divider>

      <v-list density="compact" nav class="px-2">
        <v-list-item prepend-icon="mdi-view-dashboard-outline" title="Dashboard" to="/" value="dashboard" rounded="lg" class="mb-4"></v-list-item>

        <v-list-subheader v-if="!rail" class="text-uppercase font-weight-bold text-caption text-primary mb-1">Swarm</v-list-subheader>
        <v-divider v-else class="my-3 mx-2 border-opacity-25" color="white"></v-divider>
        
        <v-list-item prepend-icon="mdi-layers-outline" title="Stacks" to="/swarm/stacks" value="stacks" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-monitor-dashboard" title="Nodes" to="/swarm/nodes" value="nodes" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-server-network" title="Services" to="/swarm/services" value="services" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-database-outline" title="Volumes" to="/swarm/volumes" value="volumes" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-docker" title="Containers" to="/swarm/containers" value="containers" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-layers-triple-outline" title="Images" to="/swarm/images" value="images" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-lan" title="Networks" to="/swarm/networks" value="networks" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-code-braces" title="Configs" to="/swarm/variables" value="variables" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-lock-outline" title="Secrets" to="/swarm/secrets" value="secrets" rounded="lg"></v-list-item>

        <v-list-subheader v-if="!rail" class="text-uppercase font-weight-bold text-caption text-primary mt-4 mb-1">GitOps</v-list-subheader>
        <v-divider v-else class="my-3 mx-2 border-opacity-25" color="white"></v-divider>
        <v-list-item prepend-icon="mdi-git" title="Repositories" to="/git/repositories" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-sync" title="Git Syncs" to="/git/syncs" rounded="lg"></v-list-item>

        <v-list-subheader v-if="!rail" class="text-uppercase font-weight-bold text-caption text-primary mt-4 mb-1">Settings</v-list-subheader>
        <v-divider v-else class="my-3 mx-2 border-opacity-25" color="white"></v-divider>
        <v-list-item prepend-icon="mdi-cog" title="Git" to="/settings/git" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-database-lock" title="Registries" to="/swarm/registries" rounded="lg"></v-list-item>
        <v-list-item prepend-icon="mdi-history" title="Events" to="/settings/events" rounded="lg"></v-list-item>
      </v-list>

      <!-- Bottom Rail Toggle for convenience -->
      <template v-slot:append v-if="!rail">
        <div class="pa-2">
          <v-btn
            block
            variant="text"
            prepend-icon="mdi-chevron-left"
            @click.stop="rail = true"
            size="small"
            color="grey"
          >
            Collapse
          </v-btn>
        </div>
      </template>
    </v-navigation-drawer>

    <v-app-bar v-if="!isAuthPage" flat class="glass-panel border-b-0" elevation="0">
      <v-app-bar-nav-icon @click.stop="rail = !rail" color="white"></v-app-bar-nav-icon>
      <v-spacer></v-spacer>
      
      <!-- Premium App Bar Actions -->
      <div class="d-flex align-center gap-2 pe-4">
        <v-btn icon variant="tonal" color="primary" class="me-2" size="small">
          <v-badge dot color="error">
            <v-icon>mdi-bell-outline</v-icon>
          </v-badge>
        </v-btn>
        
        <v-avatar size="36" color="primary" variant="tonal" class="cursor-pointer" style="border: 2px solid rgba(139, 92, 246, 0.5)">
          <span class="text-caption font-weight-bold">LE</span>
        </v-avatar>
      </div>
    </v-app-bar>

    <v-main class="bg-transparent">
      <v-container fluid :class="isAuthPage ? 'pa-0 fill-height justify-center align-center d-flex' : 'pa-6'">
        <router-view v-slot="{ Component }">
          <v-fade-transition mode="out-in">
            <component :is="Component" />
          </v-fade-transition>
        </router-view>
      </v-container>
    </v-main>
  </v-app>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const drawer = ref(true)
const rail = ref(true)

// Check if we are currently on an auth page (Login or Onboarding)
const isAuthPage = computed(() => {
  return route.path === '/login' || route.path === '/onboarding'
})
</script>

<style>
/* Prevent the permanent/pinned drawer from shifting the main content abruptly */
.v-navigation-drawer {
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), width 0.3s cubic-bezier(0.4, 0, 0.2, 1) !important;
}

/* Ensure the drawer stays on top */
.v-navigation-drawer--permanent {
  z-index: 100 !important;
}

/* Make V-Main transition smoother */
.v-main {
  transition: padding 0.3s cubic-bezier(0.4, 0, 0.2, 1) !important;
}

#halyard {
  min-height: 100vh;
}

/* Global Status Bar Styles for Tables */
.v-data-table .status-bar-row td:first-child {
  border-left: 3px solid var(--status-color, #757575) !important;
  padding-left: 16px !important;
}

.status-success { --status-color: #4CAF50; }
.status-error { --status-color: #F44336; }
.status-warning { --status-color: #FFC107; }
.status-info { --status-color: #2196F3; }
.status-grey { --status-color: #757575; }

/* Custom Scrollbar for premium feel */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}
::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.05);
}
::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}
::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}
</style>
