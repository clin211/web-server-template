import { useSuspenseQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { DeleteAccountForm } from "../../../features/account";
import { currentSessionQuery } from "../../../features/session";
import { SettingsPage } from "../../../shared/components/layout/SettingsPage";

export const Route = createFileRoute("/_console/settings/account")({
  head: () => ({ meta: [{ title: "账号管理 | Gin Console" }] }),
  component: AccountSettingsPage,
});

function AccountSettingsPage() {
  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery(currentSessionQuery());

  return (
    <SettingsPage
      title="账号管理"
      description="注销后，当前账号及其登录会话将不再可用。"
      tone="danger"
    >
      <DeleteAccountForm
        username={data.user.username}
        onDeleted={() => {
          queryClient.clear();
          window.location.assign("/login");
        }}
      />
    </SettingsPage>
  );
}
