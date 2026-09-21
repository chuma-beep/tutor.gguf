import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createRouter, RouterProvider } from "@tanstack/react-router";
import { rootRoute } from "@/routes/root";
import { indexRoute } from "@/routes/index";
import { docsRoute, docsSlugRoute } from "@/routes/docs";
import "@/styles.css";

const routeTree = rootRoute.addChildren([indexRoute, docsRoute, docsSlugRoute]);

const basepath = import.meta.env.BASE_URL.replace(/\/$/, "");

const router = createRouter({ routeTree, basepath, defaultPreload: false });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
);
