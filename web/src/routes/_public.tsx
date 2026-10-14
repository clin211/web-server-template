import { Outlet, createFileRoute } from "@tanstack/react-router";
import { RoutePending } from "../shared/components/feedback/RoutePending";

export const Route = createFileRoute("/_public")({
  pendingComponent: RoutePending,
  component: PublicLayout,
});

function PublicLayout() {
  return <Outlet />;
}
