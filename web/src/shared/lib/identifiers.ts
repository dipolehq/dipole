/**
 * Shared helpers for converting between the backend's flat identifier
 * representation and the form's grouped representation.
 *
 * Backend storage (and API payloads) use a flat list of {type, value} pairs.
 * The UI groups these by type: values sharing a type are OR'd together, while
 * distinct types are AND'd together (see ClientWorkloadForm). Every call site
 * that hydrates identifiers from the API MUST use `groupIdentifiers` so the
 * AND/OR structure is derived consistently — grouping this by hand at each
 * call site is what previously caused values of the same type to render as
 * separate AND'd entries instead of OR'd values within one entry.
 */

export interface FlatIdentifier {
  type: string
  value: string
}

export interface GroupedIdentifier {
  type: string
  values: string[]
}

/** Backend stores flat {type,value} pairs; the UI groups values under a type. */
export function groupIdentifiers(flat: FlatIdentifier[]): GroupedIdentifier[] {
  const grouped: GroupedIdentifier[] = []
  const indexByType = new Map<string, number>()
  for (const { type, value } of flat) {
    const existingIndex = indexByType.get(type)
    const entry = existingIndex === undefined ? undefined : grouped[existingIndex]
    if (entry) {
      entry.values.push(value)
    } else {
      indexByType.set(type, grouped.length)
      grouped.push({ type, values: [value] })
    }
  }
  return grouped.length > 0 ? grouped : [{ type: '', values: [''] }]
}

export function flattenIdentifiers(groups: GroupedIdentifier[]): FlatIdentifier[] {
  const flat: FlatIdentifier[] = []
  for (const group of groups) {
    if (!group.type) continue
    for (const value of group.values) {
      if (value.trim()) flat.push({ type: group.type, value: value.trim() })
    }
  }
  return flat
}
