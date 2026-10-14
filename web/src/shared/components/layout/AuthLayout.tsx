import type { ReactNode } from "react";

type AuthLayoutProps = Readonly<{
  eyebrow: string;
  title: string;
  description: string;
  children: ReactNode;
}>;

export function AuthLayout({ eyebrow, title, description, children }: AuthLayoutProps) {
  return (
    <main className="min-h-[100dvh] bg-[#f5f7fa] lg:grid lg:grid-cols-[minmax(0,1.18fr)_minmax(26rem,0.82fr)]">
      <section
        aria-labelledby="auth-heading"
        className="order-2 flex min-h-[100dvh] items-center bg-[#fcfdff] px-6 py-10 sm:px-10 lg:border-l lg:border-[#e4e8ef] lg:px-14 xl:px-20"
      >
        <div className="mx-auto w-full max-w-[430px]">
          <div className="mb-16">
            <Brand dark />
          </div>

          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.16em] text-console-accent">
              {eyebrow}
            </p>
            <h2
              id="auth-heading"
              className="mt-4 text-[2.15rem] font-semibold tracking-[-0.03em] text-console-ink sm:text-5xl"
            >
              {title}
            </h2>
            <p className="mt-4 max-w-sm text-[0.95rem] leading-7 text-console-muted">
              {description}
            </p>
          </div>

          {children}
        </div>
      </section>

      <section
        aria-label="Gin Console 品牌信息"
        className="order-1 relative hidden min-h-[100dvh] overflow-hidden bg-[#e9edf3] lg:block"
      >
        <div aria-hidden="true" className="absolute inset-0 bg-[#e9edf3]" />
        <div
          aria-hidden="true"
          className="absolute -right-[18%] -top-[22%] h-[118%] w-[112%] rounded-[48%] bg-[#f8fafc]"
        />
        <div
          aria-hidden="true"
          className="absolute bottom-[9%] left-[10%] h-px w-[30%] bg-[#cbd3df]"
        />

        <div className="relative flex min-h-[100dvh] flex-col justify-between px-12 py-12 xl:px-20">
          <div className="flex items-center justify-between gap-6">
            <Brand dark />
            <span className="text-[0.65rem] font-semibold uppercase tracking-[0.2em] text-[#8c98aa]">
              Control / 01
            </span>
          </div>

          <div className="mx-auto w-full max-w-[560px] py-8">
            <div className="mb-8 flex items-center gap-3 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-[#6e7c91]">
              <span aria-hidden="true" className="h-px w-8 bg-[#8da2bd]" />
              Operations workspace
            </div>
            <h1 className="mb-8 text-3xl font-semibold tracking-[-0.03em] text-[#14243a] xl:text-4xl">
              Gin Enterprise Console
            </h1>
            <img
              src="/console-login-art.svg"
              alt="Gin Console 运营工作台的抽象图示"
              className="block max-h-[48vh] w-full object-contain"
            />
            <p className="mt-8 max-w-md text-sm leading-6 text-[#6e7c91]">
              将身份、数据与日常操作收拢到一处，保持团队在同一个清晰的工作节奏里。
            </p>
          </div>

          <div className="flex items-center justify-between gap-6 border-t border-[#cbd3df] pt-5 text-xs text-[#7e8b9e]">
            <span>© 2026 Gin Enterprise</span>
            <span>Private workspace</span>
          </div>
        </div>
      </section>
    </main>
  );
}

function Brand({ dark = false }: Readonly<{ dark?: boolean }>) {
  return (
    <div className="flex items-center gap-3">
      <span className="grid size-10 place-items-center rounded-lg bg-console-accent text-sm font-bold text-white shadow-sm">
        GC
      </span>
      <div>
        <p className={`text-sm font-semibold ${dark ? "text-console-ink" : "text-white"}`}>
          Gin Console
        </p>
        <p className={`mt-0.5 text-xs ${dark ? "text-console-muted" : "text-slate-300"}`}>
          Enterprise workspace
        </p>
      </div>
    </div>
  );
}
