'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { apiClient } from '@/lib/api'
import { useAuthStore } from '@/lib/store'

const schema = z.object({
  email: z.string().email('Некорректный email'),
  password: z.string().min(1, 'Введите пароль'),
  tenant_slug: z.string().min(1, 'Введите slug организации'),
})

type FormData = z.infer<typeof schema>

export default function LoginPage() {
  const router = useRouter()
  const setAuth = useAuthStore((s) => s.setAuth)
  const [error, setError] = useState('')

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({
    resolver: zodResolver(schema),
  })

  const onSubmit = async (data: FormData) => {
    setError('')
    const res = await apiClient.login(data.email, data.password, data.tenant_slug)
    if (!res.ok) {
      setError(res.error ?? 'Ошибка входа')
      return
    }
    setAuth(res.data.token, res.data.user)
    router.push('/dashboard')
  }

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-white mb-2">Вход в платформу</h1>
          <p className="text-gray-400 text-sm">Введите данные вашей организации</p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="bg-gray-900 border border-white/10 rounded-2xl p-8 space-y-5">
          {error && (
            <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-lg px-4 py-3 text-sm">
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Slug организации</label>
            <input
              {...register('tenant_slug')}
              placeholder="my-company"
              className="w-full bg-gray-800 border border-white/10 rounded-lg px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-brand"
            />
            {errors.tenant_slug && <p className="text-red-400 text-xs mt-1">{errors.tenant_slug.message}</p>}
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Email</label>
            <input
              {...register('email')}
              type="email"
              placeholder="you@company.com"
              className="w-full bg-gray-800 border border-white/10 rounded-lg px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-brand"
            />
            {errors.email && <p className="text-red-400 text-xs mt-1">{errors.email.message}</p>}
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Пароль</label>
            <input
              {...register('password')}
              type="password"
              placeholder="••••••••"
              className="w-full bg-gray-800 border border-white/10 rounded-lg px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-brand"
            />
            {errors.password && <p className="text-red-400 text-xs mt-1">{errors.password.message}</p>}
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full bg-brand hover:bg-brand-dark disabled:opacity-50 rounded-lg py-2.5 font-semibold text-white transition-colors"
          >
            {isSubmitting ? 'Вход...' : 'Войти'}
          </button>
        </form>

        <p className="text-center text-gray-400 text-sm mt-6">
          Нет аккаунта?{' '}
          <Link href="/register" className="text-brand hover:underline">
            Зарегистрироваться
          </Link>
        </p>
      </div>
    </div>
  )
}
