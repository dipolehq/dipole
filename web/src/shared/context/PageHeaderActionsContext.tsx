import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'

interface PageHeaderActionsContextValue {
  actions: ReactNode
  setActions: (node: ReactNode) => void
}

const PageHeaderActionsContext = createContext<PageHeaderActionsContextValue>({
  actions: null,
  setActions: () => {},
})

export function PageHeaderActionsProvider({ children }: { children: ReactNode }) {
  const [actions, setActions] = useState<ReactNode>(null)
  return (
    <PageHeaderActionsContext.Provider value={{ actions, setActions }}>
      {children}
    </PageHeaderActionsContext.Provider>
  )
}

export function usePageHeaderActionsValue(): ReactNode {
  return useContext(PageHeaderActionsContext).actions
}

export function usePageHeaderActions(node: ReactNode): void {
  const { setActions } = useContext(PageHeaderActionsContext)

  useEffect(() => {
    setActions(node)
    return () => setActions(null)
  }, [node, setActions])
}
