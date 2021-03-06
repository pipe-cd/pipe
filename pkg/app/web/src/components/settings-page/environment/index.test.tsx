import { waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setupServer } from "msw/node";
import { MemoryRouter } from "react-router-dom";
import { createReduxStore, render, screen } from "~~/test-utils";
import { Toasts } from "~/components/toasts";
import { DELETE_ENVIRONMENT_SUCCESS } from "~/constants/toast-text";
import { listApplicationsHandler } from "~/mocks/services/application";
import {
  deleteEnvironmentFailedHandler,
  deleteEnvironmentHandler,
  listEnvironmentHandler,
} from "~/mocks/services/environment";
import { dummyEnv } from "~/__fixtures__/dummy-environment";
import { SettingsEnvironmentPage } from "./index";
const server = setupServer();

beforeAll(() => {
  server.listen();
});

beforeEach(() => {
  server.use(listEnvironmentHandler, listApplicationsHandler);
});

afterEach(() => {
  server.resetHandlers();
});

afterAll(() => {
  server.close();
});

test("Deletion", async () => {
  server.use(deleteEnvironmentHandler);
  const store = createReduxStore();

  render(
    <MemoryRouter>
      <SettingsEnvironmentPage />
      <Toasts />
    </MemoryRouter>,
    {
      store,
    }
  );

  await waitFor(() => expect(screen.getByText("staging")));
  userEvent.click(screen.getByRole("button", { name: "open menu" }));
  const deleteButton = await screen.findByRole("menuitem", { name: /delete/i });
  userEvent.click(deleteButton);

  await waitFor(() => expect(screen.getByText("Deleting Environment")));
  expect(
    screen.getByRole("link", { name: /view applications/i })
  ).toHaveAttribute("href", `/applications?envId=${dummyEnv.id}`);
  userEvent.click(await screen.findByRole("button", { name: /delete/i }));

  await waitFor(() =>
    expect(screen.getByText(DELETE_ENVIRONMENT_SUCCESS)).toBeInTheDocument()
  );
});

test("Deletion failure", async () => {
  server.use(deleteEnvironmentFailedHandler);
  const store = createReduxStore();

  render(
    <MemoryRouter>
      <SettingsEnvironmentPage />
      <Toasts />
    </MemoryRouter>,
    {
      store,
    }
  );

  await waitFor(() => expect(screen.getByText("staging")));
  userEvent.click(screen.getByRole("button", { name: "open menu" }));
  const deleteButton = await screen.findByRole("menuitem", { name: /delete/i });
  userEvent.click(deleteButton);

  await waitFor(() => expect(screen.getByText("Deleting Environment")));
  userEvent.click(await screen.findByRole("button", { name: /delete/i }));

  await waitFor(() =>
    expect(screen.getByText("Error Message")).toBeInTheDocument()
  );
  expect(
    screen.queryByText(DELETE_ENVIRONMENT_SUCCESS)
  ).not.toBeInTheDocument();
});
