import { createRouter, createWebHistory } from "vue-router";
import { useUserStore } from "@/store/user";

const routes = [
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/Login.vue"),
    meta: { title: "登录" },
  },
  {
    path: "/",
    component: () => import("@/layout/BasicLayout.vue"),
    redirect: "/dashboard",
    children: [
      {
        path: "dashboard",
        name: "Dashboard",
        component: () => import("@/views/Dashboard.vue"),
        meta: { title: "实时看板", icon: "DashboardOutlined" },
      },
      {
        path: "sell",
        name: "SellTicket",
        component: () => import("@/views/SellTicket.vue"),
        meta: {
          title: "售票管理",
          icon: "TicketOutlined",
          roles: ["admin", "seller"],
        },
      },
      {
        path: "tickets",
        name: "Tickets",
        component: () => import("@/views/Tickets.vue"),
        meta: { title: "票务查询", icon: "SearchOutlined" },
      },
      {
        path: "check-records",
        name: "CheckRecords",
        component: () => import("@/views/CheckRecords.vue"),
        meta: { title: "核销记录", icon: "CheckCircleOutlined" },
      },
      {
        path: "check-in",
        name: "CheckIn",
        component: () => import("@/views/CheckIn.vue"),
        meta: { title: "入园核销", icon: "LoginOutlined", roles: ["admin"] },
      },
      {
        path: "stats",
        name: "Stats",
        component: () => import("@/views/Stats.vue"),
        meta: { title: "统计分析", icon: "BarChartOutlined" },
      },
    ],
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, from, next) => {
  const userStore = useUserStore();
  const token = localStorage.getItem("token");

  if (to.path === "/login") {
    if (token) {
      next("/");
    } else {
      next();
    }
  } else {
    if (!token) {
      next("/login");
    } else {
      if (!userStore.user) {
        const user = JSON.parse(localStorage.getItem("user") || "null");
        if (user) {
          userStore.setUser(user);
        }
      }
      next();
    }
  }
});

export default router;
