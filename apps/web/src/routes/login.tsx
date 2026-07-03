import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { Button } from "@previewroll/ui/components/button";
import { Card, CardFrame, CardPanel } from "@previewroll/ui/components/card";
import { Field, FieldControl, FieldError, FieldLabel } from "@previewroll/ui/components/field";
import { Form } from "@previewroll/ui/components/form";
import { Input } from "@previewroll/ui/components/input";
import { Alert, AlertTitle } from "@previewroll/ui/components/alert";

import type { Route } from "./+types/login";
import { useConnection } from "@/context/connection-context";
import { useAuth } from "@/context/auth-context";
import { login } from "@/services/auth";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Sign in · PreviewRoll" },
    { name: "description", content: "Sign in to your PreviewRoll account" },
  ];
}

export default function Login() {
  const { connection, setConnection } = useConnection();
  const { isAuthenticated, isLoading, checkAuth } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [server, setServer] = useState(connection.server);
  const [username, setUsername] = useState(connection.username);

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      navigate("/", { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate]);

  return (
    <div
      className="flex min-h-svh items-center justify-center bg-cover bg-[#000000] bg-center p-4"
      style={{ backgroundImage: "url(/auth-background.jpg)" }}
    >
      <CardFrame className="w-full max-w-sm">
        <Card>
          <CardPanel>
            <div className="mb-6 flex flex-col items-start gap-3">
              <img src="/full-logo.png" alt="PreviewRoll" className="w-40 h-auto" />
            </div>
            <Form
              className="space-y-5"
              onSubmit={async (event) => {
                event.preventDefault();
                setError(null);
                setLoading(true);

                const formData = new FormData(event.currentTarget);
                const serverValue = formData.get("server") as string;
                const usernameValue = formData.get("username") as string;
                const password = formData.get("password") as string;

                const newConnection = { server: serverValue, username: usernameValue };
                setConnection(newConnection);

                const result = await login(newConnection, { username: usernameValue, password });

                if (!result.ok) {
                  setError(result.error || "Login failed");
                  setLoading(false);
                  return;
                }

                await checkAuth();
                navigate("/");
              }}
            >
              <Field name="server">
                <FieldLabel htmlFor="server">Server Address</FieldLabel>
                <FieldControl
                  render={
                    <Input
                      id="server"
                      name="server"
                      type="text"
                      placeholder="https://localhost:8080"
                      value={server}
                      onChange={(e) => setServer(e.target.value)}
                      size="lg"
                      required
                    />
                  }
                />
                <FieldError />
              </Field>
              <Field name="username">
                <FieldLabel htmlFor="username">Username</FieldLabel>
                <FieldControl
                  render={
                    <Input
                      id="username"
                      name="username"
                      type="text"
                      placeholder="admin"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      size="lg"
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
                      placeholder="••••••••"
                      size="lg"
                      required
                    />
                  }
                />
                <FieldError />
              </Field>
              {error && (
                <Alert variant="error">
                  <AlertTitle>{error}</AlertTitle>
                </Alert>
              )}
              <Button
                type="submit"
                className="w-full"
                size="lg"
                disabled={loading}
                loading={loading}
              >
                Start Connection
              </Button>
            </Form>
          </CardPanel>
        </Card>
      </CardFrame>
    </div>
  );
}
