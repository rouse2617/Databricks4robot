/**
 * Mirrors webpack CopyWebpackPlugin targets so Vite serves /assets/* at repo root.
 * Run on dev server start and production build.
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')

function copyDir(src, dest) {
  if (!fs.existsSync(src)) return
  fs.mkdirSync(dest, { recursive: true })
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    const from = path.join(src, entry.name)
    const to = path.join(dest, entry.name)
    if (entry.isDirectory()) {
      copyDir(from, to)
    } else {
      fs.copyFileSync(from, to)
    }
  }
}

function mergeDir(src, dest) {
  if (!fs.existsSync(src)) return
  fs.mkdirSync(dest, { recursive: true })
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    const from = path.join(src, entry.name)
    const to = path.join(dest, entry.name)
    if (entry.isDirectory()) {
      mergeDir(from, to)
    } else {
      fs.copyFileSync(from, to)
    }
  }
}

export function syncPublicAssets() {
  const assetsDest = path.join(root, 'public', 'assets')
  fs.rmSync(assetsDest, { recursive: true, force: true })

  copyDir(path.join(root, 'src', 'assets'), assetsDest)
  mergeDir(path.join(root, 'node_modules', 'argo-ui', 'src', 'assets'), assetsDest)

  const fontsDest = path.join(assetsDest, 'fonts')
  fs.mkdirSync(fontsDest, { recursive: true })
  const faWebfonts = path.join(root, 'node_modules', '@fortawesome', 'fontawesome-free', 'webfonts')
  if (fs.existsSync(faWebfonts)) {
    for (const name of fs.readdirSync(faWebfonts)) {
      fs.copyFileSync(path.join(faWebfonts, name), path.join(fontsDest, name))
    }
  }
}

const isMain =
  process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
if (isMain) {
  syncPublicAssets()
}
