import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";

export interface HostFilesResponse {
  entries: string[];
}

export function useHostFiles(path: string) {
  return useQuery<HostFilesResponse>({
    queryKey: ["host-files", path],
    queryFn: () =>
      apiFetch<HostFilesResponse>(
        `/host-files?path=${encodeURIComponent(path)}`,
      ),
    enabled: path.length > 0 && path.startsWith("/"),
    staleTime: 30_000,
  });
}
