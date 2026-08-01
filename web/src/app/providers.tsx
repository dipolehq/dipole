import { type ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { Toaster } from 'sonner'
import { queryClient } from '@/shared/lib/query-client'
import { ResourceSetProvider } from '@/shared/context/ResourceSetContext'

interface ProvidersProps {
  children: ReactNode
}

export function Providers({ children }: ProvidersProps) {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <ResourceSetProvider>
          {children}
          <Toaster position="top-right" richColors closeButton />
        </ResourceSetProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
