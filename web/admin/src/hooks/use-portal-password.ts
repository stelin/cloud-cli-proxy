import { useMutation } from "@tanstack/react-query";
import { portalApiFetch } from "@/lib/portal-api";

export function useChangeLoginPassword() {
  return useMutation({
    mutationFn: (body: { old_password: string; new_password: string }) =>
      portalApiFetch<{ status: string }>("/change-password", {
        method: "POST",
        body: JSON.stringify(body),
      }),
  });
}
