import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { isAuthenticated } from "@/lib/auth";
import { Topbar } from "@/components/layout/topbar";

export const Route = createFileRoute("/_portal")({
  beforeLoad: () => {
    if (!isAuthenticated()) {
      throw redirect({ to: "/login" });
    }
  },
  component: PortalLayout,
});

function PortalLayout() {
  return (
    <div className="flex h-screen flex-col">
      <Topbar />
      <main className="flex-1 overflow-y-auto bg-muted/40 p-6">
        <Outlet />
      </main>
    </div>
  );
}
