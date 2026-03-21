import Link from 'next/link'

export default function HomePage() {
  return (
    <main className="min-h-screen bg-gray-950 text-white">
      {/* Hero */}
      <section className="flex flex-col items-center justify-center min-h-screen px-6 text-center">
        <div className="max-w-4xl mx-auto">
          <div className="inline-flex items-center gap-2 bg-brand/10 border border-brand/20 rounded-full px-4 py-1.5 text-brand text-sm mb-8">
            <span className="w-2 h-2 rounded-full bg-brand animate-pulse" />
            SaaS-платформа для онлайн-мероприятий
          </div>
          <h1 className="text-5xl md:text-7xl font-bold tracking-tight mb-6">
            Проводите вебинары,{' '}
            <span className="text-brand">конференции</span>{' '}
            и трансляции
          </h1>
          <p className="text-xl text-gray-400 mb-10 max-w-2xl mx-auto">
            Собственная платформа на базе LiveKit. Полный контроль над данными,
            аналитикой и брендингом. До 10&thinsp;000 зрителей в прямом эфире.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link
              href="/register"
              className="px-8 py-3 bg-brand hover:bg-brand-dark rounded-xl font-semibold text-white transition-colors"
            >
              Начать бесплатно
            </Link>
            <Link
              href="/events"
              className="px-8 py-3 bg-white/5 hover:bg-white/10 border border-white/10 rounded-xl font-semibold text-white transition-colors"
            >
              Смотреть мероприятия
            </Link>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="py-24 px-6 border-t border-white/5">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-16">
            Всё что нужно для онлайн-мероприятий
          </h2>
          <div className="grid md:grid-cols-3 gap-8">
            {features.map((f) => (
              <div key={f.title} className="p-6 bg-white/5 rounded-2xl border border-white/10">
                <div className="text-3xl mb-4">{f.icon}</div>
                <h3 className="font-semibold text-lg mb-2">{f.title}</h3>
                <p className="text-gray-400 text-sm">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>
    </main>
  )
}

const features = [
  {
    icon: '🎥',
    title: 'HD Видео и звук',
    desc: 'Трансляция на базе LiveKit — минимальная задержка, E2E-шифрование, адаптивный битрейт.',
  },
  {
    icon: '👥',
    title: 'До 10 000 зрителей',
    desc: 'Встречи для 1000 участников или трансляции на 10 000 зрителей через HLS.',
  },
  {
    icon: '🏢',
    title: 'Мультитенант',
    desc: 'Каждый клиент — изолированный тенант со своим брендингом, доменом и данными.',
  },
  {
    icon: '📊',
    title: 'Аналитика',
    desc: 'Статистика в реальном времени: присутствие, вовлечённость, воспроизведение записей.',
  },
  {
    icon: '🤖',
    title: 'AI-слой',
    desc: 'Авто-транскрипция, резюме встречи, умный поиск по записям.',
  },
  {
    icon: '🔒',
    title: 'Безопасность',
    desc: 'RBAC, JWT-аутентификация, согласие PDPA, audit log каждого действия.',
  },
]
