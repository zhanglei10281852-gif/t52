import { defineStore } from "pinia";

export const useUserStore = defineStore("user", {
  state: () => ({
    user: null,
    token: "",
  }),
  actions: {
    setUser(user) {
      this.user = user;
      localStorage.setItem("user", JSON.stringify(user));
    },
    setToken(token) {
      this.token = token;
      localStorage.setItem("token", token);
    },
    logout() {
      this.user = null;
      this.token = "";
      localStorage.removeItem("token");
      localStorage.removeItem("user");
    },
    loadFromStorage() {
      const token = localStorage.getItem("token");
      const userStr = localStorage.getItem("user");
      if (token) {
        this.token = token;
      }
      if (userStr) {
        this.user = JSON.parse(userStr);
      }
    },
  },
});
