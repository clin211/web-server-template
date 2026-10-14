import { useSuspenseQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { ProfileForm } from "../../../features/account";
import { currentSessionQuery, sessionKeys } from "../../../features/session";
import { SettingsPage } from "../../../shared/components/layout/SettingsPage";

export const Route = createFileRoute("/_console/settings/profile")({
  head: () => ({ meta: [{ title: "个人资料 | Gin Console" }] }),
  component: ProfileSettingsPage,
});

function ProfileSettingsPage() {
  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery(currentSessionQuery());
  const { user } = data;

  return (
    <SettingsPage title="个人资料" description={`账号 ID：${user.userID}`}>
      <ProfileForm
        profile={{
          username: user.username,
          nickname: user.nickname,
          email: user.email,
          phone: user.phone,
        }}
        onUpdated={() =>
          queryClient.invalidateQueries({ queryKey: sessionKeys.current(), refetchType: "active" })
        }
      />
    </SettingsPage>
  );
}
