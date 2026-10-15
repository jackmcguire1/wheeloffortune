import { createRouter, createWebHistory } from "vue-router";

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", name: "home", component: () => import("@/pages/HomePage.vue") },
    { path: "/wheels/new", name: "builder", component: () => import("@/pages/BuilderPage.vue") },
    { path: "/wheels/:id", name: "wheel", component: () => import("@/pages/WheelPage.vue"), props: true },
    { path: "/:pathMatch(.*)*", redirect: "/" },
  ],
  scrollBehavior: () => ({ top: 0 }),
});
