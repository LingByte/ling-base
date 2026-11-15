import Link from "next/link"
import { ArrowRightIcon } from "lucide-react"

import { Badge } from "@/registry/new-york-v4/ui/badge"

export function Announcement() {
  return (
    <Badge asChild variant="secondary" className="bg-muted">
      <Link href="/docs/installation">
        按需 go get，零无关 SDK <ArrowRightIcon />
      </Link>
    </Badge>
  )
}
