import { Button } from "@previewroll/ui/components/button";
import {
  Card,
  CardFrame,
  CardFrameDescription,
  CardFrameHeader,
  CardFrameTitle,
  CardPanel,
} from "@previewroll/ui/components/card";
import { Field, FieldControl, FieldError, FieldLabel } from "@previewroll/ui/components/field";
import { Form } from "@previewroll/ui/components/form";

import type { Route } from "./+types/login";
import { Input } from "@previewroll/ui/components/input";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Sign in - PreviewRoll" },
    { name: "description", content: "Sign in to your PreviewRoll account" },
  ];
}

export default function Login() {
  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <CardFrame className="w-full max-w-sm">
        <CardFrameHeader>
          <CardFrameTitle>Sign In</CardFrameTitle>
          <CardFrameDescription>Sign in to manage your deployments.</CardFrameDescription>
        </CardFrameHeader>

        <Card>
          <CardPanel>
            <Form
              className="space-y-5"
              onSubmit={(event) => {
                event.preventDefault();

                const formData = new FormData(event.currentTarget);

                console.log({
                  email: formData.get("email"),
                  password: formData.get("password"),
                });
              }}
            >
                <Field name="email">
                  <FieldLabel htmlFor="email">Email</FieldLabel>
                  <FieldControl
                    render={
                      <Input
                        id="email"
                        name="email"
                        type="email"
                        placeholder="name@example.com"
                        required
                      />
                    }
                  />
                  <FieldError />
                </Field>

                <Field name="password">
                  <FieldLabel htmlFor="password">Password</FieldLabel>
                  <FieldControl
                    render={
                      <Input
                        id="password"
                        name="password"
                        type="password"
                        placeholder="Enter your password"
                        required
                      />
                    }
                  />
                  <FieldError />
                </Field>
              <Button type="submit" className="w-full">
                Continue
              </Button>
            </Form>
          </CardPanel>
        </Card>
      </CardFrame>
    </div>
  );
}
