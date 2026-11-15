import { JetBrains_Mono, Source_Sans_3 } from "next/font/google"
import { cn } from "cn"

// Prefer fonts that resolve reliably with webpack (avoid Turbopack Geist CSS stubs).
const fontSans = Source_Sans_3({
  subsets: ["latin"],
  variable: "--font-sans",
})

const fontHeading = Source_Sans_3({
  subsets: ["latin"],
  variable: "--font-heading",
})

const fontMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-mono",
  weight: ["400", "500"],
})

export const fontVariables = cn(
  fontSans.variable,
  fontHeading.variable,
  fontMono.variable
)
