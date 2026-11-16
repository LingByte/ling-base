import type { MetadataRoute } from "next"

import { siteConfig } from "@/lib/config"
import { source } from "@/lib/source"

const isStaticDeploy = process.env.NEXT_PUBLIC_STATIC_DEPLOY === "1"

const showcaseRoutes = ["/", "/blocks", "/colors", "/create", "/sera", "/typeset"]
const chartTypes = ["area", "bar", "line", "pie", "radar", "radial", "tooltip"]

export default function sitemap(): MetadataRoute.Sitemap {
  const docUrls = source.getPages().map((page) => page.url)
  const urls = isStaticDeploy
    ? ["/", ...docUrls]
    : [
        ...showcaseRoutes,
        ...docUrls,
        ...chartTypes.map((type) => `/charts/${type}`),
      ]

  return [...new Set(urls)].map((path) => ({
    url: new URL(path, siteConfig.url).toString(),
  }))
}
