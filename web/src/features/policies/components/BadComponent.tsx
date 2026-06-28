// Simulate a cross-feature import (policies importing from identities — should be forbidden)
import { useIdentities } from '@/features/identities/hooks/useIdentities'
export function BadComponent() { return useIdentities() }
