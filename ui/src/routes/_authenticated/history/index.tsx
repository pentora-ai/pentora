import { createFileRoute } from '@tanstack/react-router'
import ComingSoon from '@/components/coming-soon'

export const Route = createFileRoute('/_authenticated/history/')({
  component: ComingSoon,
})
