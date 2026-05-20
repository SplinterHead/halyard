import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import vuetify from './plugins/vuetify'
import router from './router'
import VueApexCharts from "vue3-apexcharts";

// Intercept all API fetches to inject the Authorization header and handle 401 redirects
const originalFetch = window.fetch;
window.fetch = async (input, init) => {
  const token = localStorage.getItem("halyard_token");
  if (token) {
    init = init || {};
    init.headers = init.headers || {};
    if (init.headers instanceof Headers) {
      init.headers.set("Authorization", `Bearer ${token}`);
    } else if (Array.isArray(init.headers)) {
      init.headers.push(["Authorization", `Bearer ${token}`]);
    } else {
      (init.headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
    }
  }

  const response = await originalFetch(input, init);

  // If 401 Unauthorized occurs on an API call (excluding login/status check), redirect to login
  if (response.status === 401) {
    const urlStr = typeof input === 'string' ? input : (input as Request).url || '';
    if (!urlStr.includes("/api/auth/login") && !urlStr.includes("/api/auth/status")) {
      localStorage.removeItem("halyard_token");
      localStorage.removeItem("halyard_user");
      window.location.href = "/login";
    }
  }

  return response;
};

const app = createApp(App)
app.use(vuetify)
app.use(router)
app.use(VueApexCharts)
app.mount('#app')
