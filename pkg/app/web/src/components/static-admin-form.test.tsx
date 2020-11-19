import userEvent from "@testing-library/user-event";
import React from "react";
import {
  createStore,
  render,
  screen,
  act,
  waitForElementToBeRemoved,
  waitFor,
} from "../../test-utils";
import { server } from "../mocks/server";
import { updateStaticAdmin } from "../modules/project";
import { StaticAdminForm } from "./static-admin-form";

beforeAll(() => {
  server.listen();
});

afterEach(() => {
  server.resetHandlers();
});

afterAll(() => {
  server.close();
});

it("should shows current username", () => {
  render(<StaticAdminForm />, {
    initialState: {
      project: {
        username: "pipe-user",
        staticAdminDisabled: false,
      },
    },
  });

  expect(screen.getByText("pipe-user")).toBeInTheDocument();
});

it("should dispatch action that update static admin when input fields and click submit button", async () => {
  const store = createStore({
    project: {
      username: "pipe-user",
      staticAdminDisabled: false,
    },
  });

  render(<StaticAdminForm />, {
    store,
  });

  userEvent.click(
    screen.getByRole("button", { name: "edit static admin user" })
  );

  await waitFor(() => screen.getByText("Edit Static Admin"));

  userEvent.type(screen.getByRole("textbox", { name: /username/i }), "-new");
  userEvent.type(screen.getByLabelText(/password/i), "new-password");

  act(() => {
    userEvent.click(screen.getByRole("button", { name: /save/i }));
  });

  await waitForElementToBeRemoved(() => screen.getByText("Edit Static Admin"));

  expect(store.getActions()).toMatchObject([
    {
      type: updateStaticAdmin.pending.type,
      meta: {
        arg: {
          username: "pipe-user-new",
          password: "new-password",
        },
      },
    },
    {
      type: updateStaticAdmin.fulfilled.type,
    },
    {},
    {},
    {},
  ]);
});
