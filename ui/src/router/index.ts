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
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;
