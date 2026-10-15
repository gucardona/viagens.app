export type Kind = 'flight' | 'lodging' | 'car' | 'transport' | 'restaurant' | 'activity' | 'event' | 'note'

export type Trip = {
  id: number
  name: string
  destination: string
  start_date: string
  end_date: string
  notes: string
  created_at: string
  updated_at: string
  countries?: string[]
  share_token?: string
}

export type Item = {
  id: number
  trip_id: number
  kind: Kind
  title: string
  start_date: string
  start_time: string
  end_date: string
  end_time: string
  address: string
  lat: number | null
  lng: number | null
  confirmation_code: string
  notes: string
  ticket: '' | 'needed' | 'bought'
  time_approx: boolean
  country_code: string
  area: string
  created_at: string
  updated_at: string
}

export type Attachment = {
  id: number
  trip_id: number
  item_id: number | null
  filename: string
  content_type: string
  size: number
  created_at: string
}

export type TripDetail = { trip: Trip; items: Item[]; attachments: Attachment[] }

export type Expense = {
  id: number
  trip_id: number
  item_id: number | null
  title: string
  amount_cents: number
  currency: string
  category: import('./money').Category
  paid_by: number
  payer_share: number
  date: string
  rate: number | null
  created_at: string
  brl_cents?: number | null
}

export type ExpenseSummary = {
  expenses: Expense[]
  total_cents: number
  by_category: Record<string, number>
  by_day: Record<string, number>
  missing_rate: number
  balance: { from: string; to: string; cents: number } | null
}

export type User = { id: number; name: string }
export type CheckItem = { id: number; text: string; done: boolean }
