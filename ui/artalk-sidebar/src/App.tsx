import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { Toast } from '@heroui/react'
import { ConfirmProvider } from '@/components/ui/ConfirmProvider'
import { useLocale } from '@/lib/i18n'
import { router } from '@/router'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 15_000,
      refetchOnWindowFocus: false,
    },
  },
})

export function App() {
  const locale = useLocale()

  return (
    <QueryClientProvider client={queryClient}>
      <ConfirmProvider>
        <Toast.Provider />
        {/* Route components are memoized on their id: a remount is the one reliable
          way to redraw every page in the new language. */}
        <RouterProvider key={locale} router={router} />
      </ConfirmProvider>
    </QueryClientProvider>
  )
}
