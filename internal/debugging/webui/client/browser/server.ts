// Browser assertions run against a fresh source build. Keep the compiler out
// of Chromium's lifetime instead of retaining Vite's transform graph beside it.
import { createReadStream } from "node:fs"
import { access, rm, stat } from "node:fs/promises"
import { createServer } from "node:http"
import { extname, resolve, sep } from "node:path"
import { fileURLToPath } from "node:url"

export default async function serve() {
  const root = fileURLToPath(new URL("./.build", import.meta.url))
  await access(resolve(root, "index.html"))
  const contentTypes: Record<string, string> = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".png": "image/png", ".svg": "image/svg+xml" }
  const server = createServer(async (request, response) => {
    try {
      const pathname = decodeURIComponent(new URL(request.url ?? "/", "http://127.0.0.1").pathname)
      const path = resolve(root, `.${pathname === "/" || pathname === "/logs" ? "/index.html" : pathname}`)
      if (!path.startsWith(root + sep) || !(await stat(path)).isFile()) {
        response.writeHead(404).end()
        return
      }
      response.writeHead(200, { "Content-Type": contentTypes[extname(path)] ?? "application/octet-stream", "Cache-Control": "no-store" })
      createReadStream(path).on("error", () => response.destroy()).pipe(response)
    } catch {
      response.writeHead(404).end()
    }
  })
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject)
    server.listen(4179, "127.0.0.1", resolve)
  })
  return async () => {
    const closed = new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()))
    server.closeAllConnections()
    await closed
    await rm(root, { recursive: true, force: true })
  }
}
