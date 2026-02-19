import { useEffect, useState } from "react";
import type { Router } from "@gapp/client";

export function useCurrentRoute<Metadata>(router: Router<Metadata>): Metadata {
  const [metadata, setMetadata] = useState<Metadata>(router.current());

  useEffect(() => {
    return router.onNavigate(setMetadata);
  }, [router]);

  return metadata;
}
