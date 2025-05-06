import { createFileRoute } from '@tanstack/react-router'
import SignIn2 from '@/features/auth/sign-in/login'

export const Route = createFileRoute('/(auth)/login')({
  component: SignIn2,
})
