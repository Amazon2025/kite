import { useQuery } from "@tanstack/react-query";

import { api, unwrap } from "@/api/client";

/** useStorage says how much the project keeps on disk, as the site measures it. */
export function useStorage() {
  return useQuery({
    queryKey: ["storage"],
    queryFn: async () => unwrap(await api.GET("/storage")),
  });
}
