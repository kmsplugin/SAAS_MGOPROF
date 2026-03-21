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
  password: z.string().min(8, 'Минимум 8 символов'),
  first_name: z.string().min(1, 'Введите имя'),
  last_name: z.string().min(1, 'Введите фамилию'),
  tenant_slug: z.string().min(1, 'Введите slug организации'),
  consent_given: z.boolean().refine((v) => v === true, 'Необходимо согласие'),
})

type FormData = z.infer<typeof schema>

export default function RegisterPage() {
  const router = useRouter()
  const setAuth = useAuthStore((s) => s.setAuth)
  const [error, setError] = useState('')

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { consent_given: false },
  })

  const onSubmit = async (data: FormData) => {
    setError('')
    const res = await apiClient.register({
      email: data.email,
      password: data.password,
      first_name: data.first_name,
      last_name: data.last_name,
      tenant_slug: data.tenant_slug,
      consent_given: data.consent_given,
    })
    if (!res.ok) {
      setError(res.error ?? 'Ошибка регистрации')
      return
    }
    setAuth(res.data.token, res.data.user)
    router.push('/dashboard')
  }

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center px-4 py-12">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-white mb-2">Регистрация</h1>
          <p className="text-gray-400 text-sm">Создайте аккаунт в вашей организации</p>
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

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-gray-400 mb-1.5">Имя</label>
              <input
                {...register('first_name')}
                placeholder="Иван"
                className="w-full bg-gray-800 border border-white/10 rounded-lg px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-brand"
              />
              {errors.first_name && <p className="text-red-400 text-xs mt-1">{errors.first_name.message}</p>}
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1.5">Фамилия</label>
              <input
                {...register('last_name')}
                placeholder="Иванов"
                className="w-full bg-gray-800 border border-white/10 rounded-lg px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-brand"
              />
              {errors.last_name && <p className="text-red-400 text-xs mt-1">{errors.last_name.message}</p>}
            </div>
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
              placeholder="Минимум 8 символов"
              className="w-full bg-gray-800 border border-white/10 rounded-lg px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-brand"
            />
            {errors.password && <p className="text-red-400 text-xs mt-1">{errors.password.message}</p>}
          </div>

          <label className="flex items-start gap-3 cursor-pointer">
            <input
              {...register('consent_given')}
              type="checkbox"
              className="mt-0.5 accent-brand"
            />
            <span className="text-sm text-gray-400">
              Я даю согласие на обработку персональных данных в соответствии с политикой конфиденциальности
            </span>
          </label>
          {errors.consent_given && <p className="text-red-400 text-xs">{errors.consent_given.message}</p>}

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full bg-brand hover:bg-brand-dark disabled:opacity-50 rounded-lg py-2.5 font-semibold text-white transition-colors"
          >
            {isSubmitting ? 'Регистрация...' : 'Зарегистрироваться'}
          </button>
        </form>

        <p className="text-center text-gray-400 text-sm mt-6">
          Уже есть аккаунт?{' '}
          <Link href="/login" className="text-brand hover:underline">
            Войти
          </Link>
        </p>
      </div>
    </div>
  )
}
