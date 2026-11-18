<template>
  <v-app id="halyard">
    <v-navigation-drawer
      v-if="!isAuthPage"
      v-model="drawer"
      :rail="!mobile && rail"
      :permanent="!mobile"
      :temporary="mobile"
      rail-width="56"
      width="220"
      app
      class="app-nav-drawer"
      elevation="0"
    >
      <v-list-item
        class="py-3"
        link
        @click.stop="mobile ? (drawer = false) : (rail = !rail)"
      >
        <template v-slot:prepend>
          <img src="/logo.png" alt="Halyard" class="rail-brand" />
        </template>
        <v-list-item-title
          v-if="mobile || !rail"
          class="fw-600 ms-3 text-1"
          style="font-size: 14px;"
        >
          Halyard
        </v-list-item-title>
      </v-list-item>

      <v-divider style="border-color: var(--border-2); opacity: 1;"></v-divider>

      <v-list density="compact" nav class="px-2 py-2">
        <v-list-item
          prepend-icon="mdi-view-dashboard-outline"
          title="Dashboard"
          to="/"
          value="dashboard"
          rounded="md"
          class="mb-2"
        ></v-list-item>

        <v-list-subheader v-if="mobile || !rail" class="text-uppercase mb-1">Swarm</v-list-subheader>
        <v-divider v-else class="my-2 mx-2" style="border-color: var(--border-1); opacity: 1;"></v-divider>

        <v-list-item prepend-icon="mdi-sitemap-outline" title="Visualizer" to="/swarm/visualizer" value="visualizer" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-monitor-dashboard" title="Nodes" to="/swarm/nodes" value="nodes" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-layers-outline" title="Stacks" to="/swarm/stacks" value="stacks" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-server-network" title="Services" to="/swarm/services" value="services" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-docker" title="Containers" to="/swarm/containers" value="containers" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-layers-triple-outline" title="Images" to="/swarm/images" value="images" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-database-outline" title="Volumes" to="/swarm/volumes" value="volumes" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-lan" title="Networks" to="/swarm/networks" value="networks" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-code-braces" title="Configs" to="/swarm/variables" value="variables" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-lock-outline" title="Secrets" to="/swarm/secrets" value="secrets" rounded="md"></v-list-item>

        <v-list-subheader v-if="mobile || !rail" class="text-uppercase mt-3 mb-1">GitOps</v-list-subheader>
        <v-divider v-else class="my-2 mx-2" style="border-color: var(--border-1); opacity: 1;"></v-divider>

        <v-list-item prepend-icon="mdi-git" title="Repositories" to="/git/repositories" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-sync" title="Git Syncs" to="/git/syncs" rounded="md"></v-list-item>

        <v-list-subheader v-if="mobile || !rail" class="text-uppercase mt-3 mb-1">Settings</v-list-subheader>
        <v-divider v-else class="my-2 mx-2" style="border-color: var(--border-1); opacity: 1;"></v-divider>

        <v-list-item prepend-icon="mdi-cog" title="Settings" to="/settings" rounded="md"></v-list-item>
        <v-list-item prepend-icon="mdi-database-lock" title="Registries" to="/swarm/registries" rounded="md"></v-list-item>
      </v-list>

      <template v-slot:append>
        <div v-if="mobile || !rail" class="pa-3 text-center">
          <span class="mono text-3">v{{ pkg.version }}</span>
        </div>
      </template>
    </v-navigation-drawer>

    <v-app-bar v-if="!isAuthPage" flat height="48" class="app-top-bar" elevation="0">
      <div class="d-flex align-center ps-3" style="gap: 10px;">
        <button
          type="button"
          class="topbar-icon-btn"
          aria-label="Toggle navigation"
          @click="toggleNav"
        >
          <v-icon size="18">{{ !mobile && !rail ? 'mdi-menu-open' : 'mdi-menu' }}</v-icon>
        </button>

        <router-link v-if="mobile" to="/" class="mobile-brand-link">
          <img src="/logo.png" alt="Halyard" class="rail-brand" style="width: 24px; height: 24px;" />
          <span class="fw-600 text-1" style="font-size: 14px;">Halyard</span>
        </router-link>
      </div>

      <v-spacer></v-spacer>

      <div class="d-flex align-center pe-4">
        <v-menu>
          <template v-slot:activator="{ props }">
            <button
              v-bind="props"
              type="button"
              class="avatar-btn"
              aria-label="User menu"
            >
              {{ userInitials }}
            </button>
          </template>
          <v-list class="mt-2 app-menu-list" density="compact" elevation="0">
            <v-list-item to="/settings/profile" prepend-icon="mdi-account-cog-outline" value="profile">
              <v-list-item-title style="font-size: 12.5px;">Profile Settings</v-list-item-title>
            </v-list-item>
            <v-list-item
              :prepend-icon="isDark ? 'mdi-weather-sunny' : 'mdi-weather-night'"
              value="theme-toggle"
              @click="toggleTheme()"
            >
              <v-list-item-title style="font-size: 12.5px;">
                {{ isDark ? 'Light Theme' : 'Dark Theme' }}
              </v-list-item-title>
            </v-list-item>
            <v-divider class="my-1" style="border-color: var(--border-2); opacity: 1;"></v-divider>
            <v-list-item prepend-icon="mdi-logout" value="logout" @click="handleLogout">
              <v-list-item-title class="text-err" style="font-size: 12.5px;">Logout</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>
      </div>
    </v-app-bar>

    <v-main>
      <div v-if="isAuthPage" class="auth-page-host">
        <router-view />
      </div>
      <div v-else class="view-container">
        <router-view v-slot="{ Component, route: currentRoute }">
          <component :is="Component" :key="currentRoute.path" />
        </router-view>
      </div>
    </v-main>

    <v-bottom-navigation
      v-if="!isAuthPage && mobile"
      grow
      height="56"
      class="app-bottom-nav"
      elevation="0"
    >
      <v-btn to="/" exact>
        <v-icon size="18">mdi-view-dashboard-outline</v-icon>
        <span>Cluster</span>
      </v-btn>
      <v-btn to="/swarm/stacks">
        <v-icon size="18">mdi-layers-outline</v-icon>
        <span>Stacks</span>
      </v-btn>
      <v-btn to="/git/syncs">
        <v-icon size="18">mdi-sync</v-icon>
        <span>GitOps</span>
      </v-btn>
      <v-btn to="/swarm/nodes">
        <v-icon size="18">mdi-monitor-dashboard</v-icon>
        <span>Nodes</span>
      </v-btn>
      <v-btn @click="drawer = true">
        <v-icon size="18">mdi-menu</v-icon>
        <span>More</span>
      </v-btn>
    </v-bottom-navigation>
  </v-app>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useDisplay, useTheme as useVuetifyTheme } from 'vuetify';
import { useTheme } from './theme';
import pkg from '../package.json';

const route = useRoute();
const router = useRouter();
const { mobile } = useDisplay();
const vuetifyTheme = useVuetifyTheme();
const { isDark, toggleTheme } = useTheme(vuetifyTheme);

const drawer = ref(!mobile.value);
const rail = ref(true);
const userInitials = ref('??');

const isAuthPage = computed(() => {
  return route.path === '/login' || route.path === '/onboarding';
});

const toggleNav = () => {
  if (mobile.value) {
    drawer.value = !drawer.value;
  } else {
    rail.value = !rail.value;
  }
};

watch(mobile, (isMob) => {
  drawer.value = !isMob;
});

const loadUser = () => {
  const userStr = localStorage.getItem('halyard_user');
  if (userStr) {
    try {
      const user = JSON.parse(userStr);
      if (user.real_name) {
        userInitials.value = user.real_name
          .split(' ')
          .map((n: string) => n[0])
          .join('')
          .substring(0, 2)
          .toUpperCase();
      } else if (user.username) {
        userInitials.value = user.username.substring(0, 2).toUpperCase();
      }

      if (user.preferences) {
        const root = document.documentElement;
        for (const [key, value] of Object.entries(user.preferences)) {
          if (value && key.startsWith('ansi-')) {
            root.style.setProperty(`--${key}`, value as string);
          }
        }
      }
    } catch (e) {
      console.error('Failed to parse user', e);
    }
  }
};

const handleLogout = () => {
  localStorage.removeItem('halyard_token');
  localStorage.removeItem('halyard_user');
  router.push('/login');
};

onMounted(() => {
  if (!isAuthPage.value) {
    loadUser();
  }
});

watch(isAuthPage, (newVal) => {
  if (!newVal) {
    loadUser();
  }
});
</script>

<style scoped>
.topbar-icon-btn {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  border: 0;
  background: transparent;
  color: var(--text-2);
  display: grid;
  place-items: center;
  cursor: pointer;
}

.topbar-icon-btn:hover {
  background: var(--surface-2);
  color: var(--text-1);
}

.mobile-brand-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
}

.avatar-btn {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--surface-2);
  border: 1px solid var(--border-1);
  color: var(--text-1);
  font-size: 11px;
  font-weight: 600;
  display: grid;
  place-items: center;
  cursor: pointer;
}

.avatar-btn:hover {
  border-color: var(--accent);
}

.auth-page-host {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-0);
  padding: 20px;
}
</style>
