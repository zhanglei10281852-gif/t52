import { createApp } from "vue";
import { createPinia } from "pinia";
import Antd from "ant-design-vue";
import "ant-design-vue/dist/reset.css";
import * as Icons from "@ant-design/icons-vue";
import App from "./App.vue";
import router from "./router";
import "./style.css";

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(Antd);

for (const key in Icons) {
  app.component(key, Icons[key]);
}

app.mount("#app");
