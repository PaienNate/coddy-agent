import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { ProjectRail } from "./ProjectRail";
import type { Project } from "./types";

afterEach(() => cleanup());

const project = (id: string, name: string): Project => ({
  id,
  name,
  path: `/work/${id}`,
  order: 0,
  createdAt: "",
  updatedAt: "",
});

test("the project rail lists the projects and marks the open one", () => {
  render(
    <ProjectRail
      projects={[project("a", "Alpha App"), project("b", "Beta")]}
      activeProjectId="b"
      onSelect={() => {}}
      onOpenProjects={() => {}}
    />,
  );
  expect(screen.getAllByRole("button")).toHaveLength(3);
  expect(
    screen.getByLabelText("Switch to Beta").getAttribute("aria-current"),
  ).toBe("page");
  expect(screen.getByText("AA")).toBeInTheDocument();
  expect(screen.getByText("BE")).toBeInTheDocument();
});

test("picking another project and the all-projects button call their handlers", () => {
  const onSelect = vi.fn();
  const onOpenProjects = vi.fn();
  render(
    <ProjectRail
      projects={[project("a", "Alpha"), project("b", "Beta")]}
      activeProjectId="a"
      onSelect={onSelect}
      onOpenProjects={onOpenProjects}
    />,
  );
  fireEvent.click(screen.getByLabelText("Switch to Beta"));
  expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: "b" }));
  fireEvent.click(screen.getByLabelText("All projects"));
  expect(onOpenProjects).toHaveBeenCalledTimes(1);
});
