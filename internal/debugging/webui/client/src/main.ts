import { createApp } from "vue"

import "./style.css"
import { rootComponent } from "./root_component"

createApp(rootComponent(location.pathname)).mount("#app")
