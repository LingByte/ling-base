import { RootProvider } from 'fumadocs-ui/provider/next';
import './global.css';
import { Inter } from 'next/font/google';
import { PlaygroundProvider } from '@/components/playground/drawer-context';
import { PlaygroundDrawer } from '@/components/playground/PlaygroundDrawer';

const inter = Inter({
  subsets: ['latin'],
});

export default function Layout({ children }: LayoutProps<'/'>) {
  return (
    <html lang="en" className={inter.className} suppressHydrationWarning>
      <body className="flex flex-col min-h-screen">
        <RootProvider>
          <PlaygroundProvider>
            {children}
            <PlaygroundDrawer />
          </PlaygroundProvider>
        </RootProvider>
      </body>
    </html>
  );
}
