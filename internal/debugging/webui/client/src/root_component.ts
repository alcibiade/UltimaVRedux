import App from "./App.vue"
import LogWindow from "./LogWindow.vue"

export function rootComponent(pathname: string): typeof App | typeof LogWindow {
  return pathname === "/logs" ? LogWindow : App
}
