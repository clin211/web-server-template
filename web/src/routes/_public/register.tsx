import { createFileRoute } from "@tanstack/react-router";
import { RegistrationForm } from "../../features/registration";
import { AuthLayout } from "../../shared/components/layout/AuthLayout";

export const Route = createFileRoute("/_public/register")({
  head: () => ({ meta: [{ title: "创建账号 | Gin Console" }] }),
  component: RegisterPage,
});

function RegisterPage() {
  return (
    <AuthLayout
      eyebrow="新成员注册"
      title="创建账号"
      description="建立用于访问 Gin Console 的企业账号"
    >
      <RegistrationForm />
    </AuthLayout>
  );
}
