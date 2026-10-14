import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import './index.css'
import Login from './pages/Login'
import Trips from './pages/Trips'
import TripForm from './pages/TripForm'
import Trip from './pages/Trip'
import ItemSheet from './pages/ItemSheet'
import ExpenseForm from './pages/ExpenseForm'
import { Live } from './live'

const queryClient = new QueryClient({ defaultOptions: { queries: { refetchOnWindowFocus: true, retry: false } } })

if (import.meta.env.PROD && 'serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js')

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Live />
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<Trips />} />
          <Route path="/viagens/nova" element={<TripForm />} />
          <Route path="/viagens/:id" element={<Trip />} />
          <Route path="/viagens/:id/editar" element={<TripForm />} />
          <Route path="/viagens/:id/itens/:itemId" element={<ItemSheet />} />
          <Route path="/viagens/:id/gastos/:expenseId" element={<ExpenseForm />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)
