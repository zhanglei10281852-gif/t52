import axios from "axios";
import { message } from "ant-design-vue";
import router from "@/router";

const request = axios.create({
  baseURL: "/api",
  timeout: 10000,
});

request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem("token");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  },
);

request.interceptors.response.use(
  (response) => {
    return response.data;
  },
  (error) => {
    if (error.response) {
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        message.error("登录已过期，请重新登录");
        router.push("/login");
      } else {
        message.error(error.response.data?.error || "请求失败");
      }
    } else {
      message.error("网络错误，请稍后重试");
    }
    return Promise.reject(error);
  },
);

export default request;
