"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { signIn } from "next-auth/react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

const schema = z.object({
  email: z.string().min(1, "E-posta gerekli"),
  password: z.string().min(1, "Parola gerekli"),
});
type Values = z.infer<typeof schema>;

export function LoginForm() {
  const router = useRouter();
  const [serverError, setServerError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "" },
  });

  async function onSubmit(values: Values) {
    setServerError(null);
    const res = await signIn("credentials", { ...values, redirect: false });
    if (res?.error) {
      setServerError("E-posta veya parola hatalı.");
      return;
    }
    router.push("/");
    router.refresh();
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 px-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-zinc-800 to-zinc-950 text-white shadow-lg">
            <svg
              className="h-6 w-6"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <circle cx="12" cy="5" r="2.5" />
              <circle cx="6" cy="18" r="2.5" />
              <circle cx="18" cy="18" r="2.5" />
              <line x1="12" y1="7.5" x2="6" y2="15.5" />
              <line x1="12" y1="7.5" x2="18" y2="15.5" />
              <line x1="6" y1="18" x2="18" y2="18" />
            </svg>
          </div>
          <div className="text-center">
            <h1 className="text-xl font-bold tracking-tight text-zinc-950">BrainMeta</h1>
            <p className="mt-1 text-xs font-medium text-zinc-500">
              Karar Konsolu — panele giriş
            </p>
          </div>
        </div>

        <form
          onSubmit={handleSubmit(onSubmit)}
          className="rounded-2xl border border-zinc-200/70 bg-white p-6 shadow-sm"
        >
          <div className="space-y-4">
            <div>
              <label
                htmlFor="email"
                className="mb-1.5 block text-xs font-semibold text-zinc-700"
              >
                E-posta
              </label>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                placeholder="admin@disci.local"
                {...register("email")}
              />
              {errors.email && (
                <p className="mt-1 text-[11px] font-medium text-red-500">
                  {errors.email.message}
                </p>
              )}
            </div>
            <div>
              <label
                htmlFor="password"
                className="mb-1.5 block text-xs font-semibold text-zinc-700"
              >
                Parola
              </label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                placeholder="••••••••"
                {...register("password")}
              />
              {errors.password && (
                <p className="mt-1 text-[11px] font-medium text-red-500">
                  {errors.password.message}
                </p>
              )}
            </div>
            {serverError && (
              <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-[11px] font-medium text-red-600">
                {serverError}
              </div>
            )}
            <Button type="submit" disabled={isSubmitting} className="w-full">
              {isSubmitting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" /> Giriş yapılıyor…
                </>
              ) : (
                "Giriş yap"
              )}
            </Button>
          </div>
        </form>
        <p className="mt-4 text-center text-[11px] text-zinc-400">
          Geliştirme girişi: admin@disci.local / admin1234
        </p>
      </div>
    </div>
  );
}
