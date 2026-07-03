import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { Button } from "@previewroll/ui/components/button";
import {
  Dialog,
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogPanel,
  DialogPopup,
  DialogTitle,
  DialogTrigger,
} from "@previewroll/ui/components/dialog";
import {
  Field,
  FieldControl,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@previewroll/ui/components/field";
import { Input } from "@previewroll/ui/components/input";
import {
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
} from "@previewroll/ui/components/select";
import { Alert, AlertTitle } from "@previewroll/ui/components/alert";
import { Form } from "@previewroll/ui/components/form";
import { useConnection } from "@/context/connection-context";
import { createProject } from "@/services/projects";
import { detectVcsProvider } from "@/lib/utils";
import slugify from "slugify";

interface CreateProjectDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const vcsProviders = {
  github: "GitHub",
  gitlab: "GitLab",
  bitbucket: "Bitbucket",
  codeberg: "Codeberg",
  other: "Other",
};

export function CreateProjectDialog({ open, onOpenChange }: CreateProjectDialogProps) {
  const { connection } = useConnection();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [repoUrl, setRepoUrl] = useState("");
  const [vcsProvider, setVcsProvider] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setName("");
      setRepoUrl("");
      setVcsProvider("");
      setError(null);
    }
  }, [open]);

  const mutation = useMutation({
    mutationFn: (values: { name: string; repo_url: string; vcs_provider: string }) =>
      createProject(connection, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
      onOpenChange(false);
    },
    onError: (err: Error) => {
      setError(err.message || "Failed to create project");
    },
  });

  const slug = name ? slugify(name, { lower: true, strict: true }) : "";

  useEffect(() => {
    const detected = detectVcsProvider(repoUrl);
    if (detected) setVcsProvider(detected);
  }, [repoUrl]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogTrigger render={<Button size="lg" />}>
        <Plus className="size-4" />
        Create project
      </DialogTrigger>
      <DialogPopup>
        <Form
          method="POST"
          className="flex flex-col flex-1"
          onFormSubmit={(formValues) => {
            setError(null);
            mutation.mutate({
              name: formValues.name as string,
              repo_url: formValues.repo_url as string,
              vcs_provider: (formValues.vcs_provider as string) || "github",
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>Create project</DialogTitle>
            <DialogDescription>Add a new project.</DialogDescription>
          </DialogHeader>
          <DialogPanel className="grid gap-4">
            <Field
              name="name"
              validationMode="onBlur"
              validate={(value) => {
                if (!value || (typeof value === "string" && value.trim() === "")) {
                  return "Please fill a project name.";
                }
                return null;
              }}
            >
              <FieldLabel htmlFor="project-name">Name</FieldLabel>
              <FieldControl
                render={
                  <Input
                    id="project-name"
                    name="name"
                    placeholder="my-project"
                    size="lg"
                    maxLength={255}
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                  />
                }
              />
              <FieldError />
              {name.length > 0 && slug !== name && (
                <FieldDescription>
                  Will be saved as <span className="text-primary">{slug}</span>
                </FieldDescription>
              )}
            </Field>
            <Field
              name="repo_url"
              validationMode="onBlur"
              validate={(value) => {
                if (!value || (typeof value === "string" && value.trim() === "")) {
                  return "Please enter a repository URL";
                }
                return null;
              }}
            >
              <FieldLabel htmlFor="project-repo">Repository URL</FieldLabel>
              <FieldControl
                render={
                  <Input
                    id="project-repo"
                    name="repo_url"
                    placeholder="https://github.com/user/repo"
                    size="lg"
                    maxLength={255}
                    value={repoUrl}
                    onChange={(e) => setRepoUrl(e.target.value)}
                  />
                }
              />
              <FieldError />
            </Field>

            <Field
              name="vcs_provider"
              validationMode="onBlur"
              validate={(value) => {
                if (!value) {
                  return "Please select a provider";
                }
                return null;
              }}
            >
              <FieldLabel htmlFor="vcs-provider">VCS Provider</FieldLabel>
              <FieldControl
                  render={
                    <Select
                        name="vcs_provider"
                        value={vcsProvider}
                        onValueChange={(value) => value && setVcsProvider(value)}
                        items={vcsProviders}
                    >
                      <SelectTrigger size="lg" className="w-full">
                        <SelectValue placeholder="Select VCS Provider" />
                      </SelectTrigger>
                      <SelectPopup>
                        <SelectItem value="github">GitHub</SelectItem>
                        <SelectItem value="gitlab">GitLab</SelectItem>
                        <SelectItem value="bitbucket">Bitbucket</SelectItem>
                        <SelectItem value="codeberg">Codeberg</SelectItem>
                        <SelectItem value="other">Other</SelectItem>
                      </SelectPopup>
                    </Select>
                  }
              />
              <FieldError />
            </Field>
            {error && (
              <Alert variant="error">
                <AlertTitle>{error}</AlertTitle>
              </Alert>
            )}
          </DialogPanel>
          <DialogFooter>
            <DialogClose
              render={<Button variant="secondary" />}
              disabled={mutation.isPending}
            >
              Cancel
            </DialogClose>
            <Button type="submit" loading={mutation.isPending} disabled={mutation.isPending}>
              Create
            </Button>
          </DialogFooter>
        </Form>
      </DialogPopup>
    </Dialog>
  );
}
