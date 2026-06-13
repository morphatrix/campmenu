export type Role = 'ADMIN' | 'COLLABORATOR' | 'USER'

type WithRole = { role?: Role } | null | undefined
export function isAdmin(u: WithRole): boolean {
  return u?.role === 'ADMIN'
}
// Staff = admin or collaborator: full content management rights.
export function isStaff(u: WithRole): boolean {
  return u?.role === 'ADMIN' || u?.role === 'COLLABORATOR'
}

// Display name shown in participant lists: nickname if set, else first+last, else email.
export function displayName(u?: { nickname?: string; firstName?: string; lastName?: string; email?: string } | null): string {
  if (!u) return ''
  if (u.nickname && u.nickname.trim()) return u.nickname.trim()
  const full = `${u.firstName ?? ''} ${u.lastName ?? ''}`.trim()
  return full || u.email || ''
}

export interface User {
  id: string
  email: string
  emailConfirmed: boolean
  role: Role
  firstName: string
  lastName: string
  nickname: string
  iban: string
  birthDate?: string | null
  shoeSize?: number | null
  weight?: number | null
  photoUrl: string
  theme: string
  colorPalette: string
  colorblindMode: boolean
  language: string
  impersonating?: boolean
}

export type TabKind = 'MENUS' | 'SHOPPING' | 'MATRIX' | 'LOCATIONS'

export interface TabArticle {
  id: string
  tabId: string
  ingredientId?: string | null
  name: string
  unit: string
  section: string
  qtyPerLevel: Record<string, number>
  quantity: number
  position: number
}

export interface TabRecipe {
  id: string
  tabId: string
  recipeId: string
  recipe?: Recipe
  section: string
  participantCount: number
  position: number
}

export interface EventTab {
  id: string
  eventId: string
  kind: TabKind
  name: string
  icon: string
  position: number
  removable: boolean
  withRecipes: boolean
  voted: boolean
  listId?: string | null
  sections: string[]
  consumptionLabels: Record<string, string>
  articles?: TabArticle[]
  recipes?: TabRecipe[]
}

export interface ProductListItem {
  id: string
  listId: string
  name: string
  unit: string
  section: string
  qtyPerLevel: Record<string, number>
  quantity: number
  position: number
}

export interface ProductList {
  id: string
  name: string
  eventId?: string | null // null = shared catalog; set = private to one event
  voted: boolean
  sections: string[]
  items?: ProductListItem[]
}

export interface EventParticipant {
  id: string
  eventId: string
  userId: string
  counted: boolean
  user?: User
}

export interface Event {
  id: string
  name: string
  startDate: string
  endDate: string
  initialParticipants: number
  photoUrl: string
  voteWeights: string
  venueAddress: string
  venueMapsUrl: string
  venuePhone: string
  venueInfo: string
  createdBy: string
  participants?: EventParticipant[]
  tabs?: EventTab[]
}

export interface Location {
  id: string
  eventId: string
  createdBy: string
  title: string
  address: string
  websiteUrl: string
  mapsUrl: string
  beds: number
  singleBeds: number
  doubleBeds: number
  toilets: number
  price: number
  phone: string
  usefulInfo: string
  description: string
  amenities: string[]
  images: string[]
  isWinner: boolean
  score: number
}

export interface LocationsResponse {
  locations: Location[]
  myVotes: Record<string, string> // rank -> locationId
  voteWeights: number[]
}

export interface Ingredient {
  id: string
  canonicalName: string
  defaultUnit: string
}

export interface RecipeIngredient {
  id: string
  recipeId: string
  ingredientId: string
  ingredient?: Ingredient
  quantity: number
  unit: string
}

export interface Recipe {
  id: string
  name: string
  basePersons: number
  coefficient: number
  photoUrl: string
  instructions: string
  kind: string
  tags: string[]
  approved: boolean
  createdBy: string
  ingredients?: RecipeIngredient[]
}

export function isCocktail(r: { tags?: string[]; kind?: string }): boolean {
  return (r.tags ?? []).some((t) => t.toLowerCase() === 'cocktail') || r.kind === 'cocktail'
}

export type MealType = 'BREAKFAST' | 'LUNCH' | 'DINNER' | 'APERITIF' | 'DESSERT'

export interface MealRecipe {
  id: string
  mealId: string
  recipeId: string
  recipe?: Recipe
  participantCount: number
  position: number
}

export interface MealRawItem {
  id: string
  mealId: string
  name: string
  quantity: number
  unit: string
}

export interface Meal {
  id: string
  eventId: string
  dayIndex: number
  type: MealType
  variant: string
  participantCount?: number | null
  recipes?: MealRecipe[]
  rawItems?: MealRawItem[]
}

export interface ShoppingLine {
  section: string
  name: string
  unit: string
  quantity: number
  ingredientId?: string | null
  source: string
  observation: string
  bought: boolean
  broughtBy?: string | null
}

export interface TabConsumption {
  id: string
  tabId: string
  articleId: string
  userId: string
  level: number
}

export interface Invite {
  id: string
  code: string
  email: string
  role: Role
  maxUses: number
  useCount: number
  revoked: boolean
  usedAt?: string | null
  expiresAt?: string | null
}

export interface SiteConfig {
  siteName: string
  logoUrl: string
  defaultTheme: string
  defaultPalette: string
}
