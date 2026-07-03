import {
  type RouteConfig,
  index,
  layout,
  route,
} from "@react-router/dev/routes";

export default [
  layout("./routes/layout.tsx", [
    index("./routes/home.tsx"),
    route("projects", "./routes/projects.tsx"),
  ]),
  route("login", "./routes/login.tsx"),
] satisfies RouteConfig;
