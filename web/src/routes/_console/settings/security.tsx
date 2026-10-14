import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { ChangePasswordForm } from "../../../features/account";
import { SettingsPage } from "../../../shared/components/layout/SettingsPage";

export const Route = createFileRoute("/_console/settings/security")({
  head: () => ({ meta: [{ title: "安全设置 | Gin Console" }] }),
  component: SecuritySettingsPage,
});

function SecuritySettingsPage() {
  const queryClient = useQueryClient();

  return (
    <SettingsPage title="安全设置" description="更新当前账号的登录密码。">
      <ChangePasswordForm
        onChanged={() => {
          queryClient.clear();
          window.location.assign("/login?reason=password-changed");
        }}
      />
    </SettingsPage>
  );
}
