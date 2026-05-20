import { createRouter, createWebHistory } from "vue-router";
import HomeView from "../views/HomeView.vue";

const routes = [
  { path: "/", name: "home", component: HomeView },
  {
    path: "/swarm/stacks",
    name: "stacks",
    component: () => import("../views/swarm/StacksView.vue"),
  },
  {
    path: "/swarm/nodes",
    name: "nodes",
    component: () => import("../views/swarm/NodesView.vue"),
  },
  {
    path: "/swarm/nodes/:id",
    name: "node-detail",
    component: () => import("../views/swarm/NodeDetailView.vue"),
  },
  {
    path: "/swarm/services",
    name: "services",
    component: () => import("../views/swarm/ServicesView.vue"),
  },
  {
    path: "/swarm/volumes",
    name: "volumes",
    component: () => import("../views/swarm/VolumesView.vue"),
  },
  {
    path: "/swarm/containers",
    name: "containers",
    component: () => import("../views/swarm/ContainersView.vue"),
  },
  {
    path: "/swarm/images",
    name: "images",
    component: () => import("../views/swarm/ImagesView.vue"),
  },
  {
    path: "/swarm/containers/:id",
    name: "container-detail",
    component: () => import("../views/swarm/ContainerDetailView.vue"),
  },
  {
    path: "/swarm/networks",
    name: "networks",
    component: () => import("../views/swarm/NetworksView.vue"),
  },
  {
    path: "/swarm/networks/:id",
    name: "network-detail",
    component: () => import("../views/swarm/NetworkDetailView.vue"),
  },
  {
    path: "/swarm/registries",
    name: "registries",
    component: () => import("../views/swarm/RegistriesView.vue"),
  },
  {
    path: "/swarm/variables",
    name: "variables",
    component: () => import("../views/swarm/VariablesView.vue"),
  },
  {
    path: "/swarm/secrets",
    name: "secrets",
    component: () => import("../views/swarm/SecretsView.vue"),
  },
  {
    path: "/swarm/stacks/:name",
    name: "stack-detail",
    component: () => import("../views/swarm/StackDetailView.vue"),
  },
  {
    path: "/swarm/services/:id",
    name: "service-detail",
    component: () => import("../views/swarm/ServiceDetailView.vue"),
  },
  {
    path: "/git/repositories",
    name: "repositories",
    component: () => import("../views/git/RepositoriesView.vue"),
  },
  {
    path: "/git/syncs",
    name: "syncs",
    component: () => import("../views/git/SyncsView.vue"),
  },
  {
    path: "/settings/git",
    name: "settings-git",
    component: () => import("../views/settings/SettingsGitView.vue"),
  },
  {
    path: "/settings/events",
    name: "events",
    component: () => import("../views/settings/SettingsEventsView.vue"),
  },
  {
    path: "/login",
    name: "login",
    component: () => import("../views/auth/LoginView.vue"),
  },
  {
    path: "/onboarding",
    name: "onboarding",
    component: () => import("../views/auth/OnboardingView.vue"),
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to, from, next) => {
  // Check auth status
  let hasUsers = false;
  try {
    const res = await fetch("/api/auth/status");
    const data = await res.json();
    hasUsers = data.has_users;
  } catch (e) {
    console.error("Failed to fetch auth status", e);
    // If the network request fails, proceed carefully.
    // If we have a local token we can still try to let them through.
    const token = localStorage.getItem("halyard_token");
    if (token) {
      return next();
    }
  }

  const token = localStorage.getItem("halyard_token");

  if (!hasUsers) {
    // If no users exist, enforce onboarding
    if (to.path !== "/onboarding") {
      next("/onboarding");
    } else {
      next();
    }
  } else {
    // Users exist
    if (!token) {
      // Unauthenticated
      if (to.path !== "/login") {
        next("/login");
      } else {
        next();
      }
    } else {
      // Authenticated
      if (to.path === "/login" || to.path === "/onboarding") {
        next("/");
      } else {
        next();
      }
    }
  }
});

export default router;
