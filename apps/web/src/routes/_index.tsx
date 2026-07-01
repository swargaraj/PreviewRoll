import { Link } from "react-router";

import { Button } from "@previewroll/ui/components/button";

import type { Route } from "./+types/_index";

export function meta({}: Route.MetaArgs) {
  return [{ title: "PreviewRoll" }, { name: "description", content: "PreviewRoll" }];
}

export default function Home() {
  return (
    <div className="container mx-auto max-w-3xl px-4 py-2">
      <h1 className="text-2xl font-bold">PreviewRoll</h1>
      <div className="mt-4">
        <Button render={<Link to="/login" />}>Sign in</Button>
      </div>
    </div>
  );
}
