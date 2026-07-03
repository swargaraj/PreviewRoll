import type { Route } from "./+types/home";

export function meta({}: Route.MetaArgs) {
  return [{ title: "PreviewRoll" }, { name: "description", content: "PreviewRoll" }];
}

export default function Home() {
  return <div />;
}
