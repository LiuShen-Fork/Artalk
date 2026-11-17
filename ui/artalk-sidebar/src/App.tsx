import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { Toast } from '@heroui/react'
import { ConfirmProvider } from '@/components/ui/ConfirmProvider'
import { useLocaleVersion } from '@/lib/i18n'
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
  useLocaleVersion()

  return (
    <QueryClientProvider client={queryClient}>
      <ConfirmProvider>
        <Toast.Provider />
        <RouterProvider router={router} />
      </ConfirmProvider>
    </QueryClientProvider>
  )
}
