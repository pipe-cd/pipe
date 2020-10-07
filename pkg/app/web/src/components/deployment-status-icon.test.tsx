import React from "react";
import { render, screen } from "../../test-utils";
import { DeploymentStatus } from "../modules/deployments";
import { StatusIcon } from "./deployment-status-icon";

test("DEPLOYMENT_CANCELLED", () => {
  render(<StatusIcon status={DeploymentStatus.DEPLOYMENT_CANCELLED} />, {});

  expect(screen.getByTestId("deployment-error-icon")).toBeInTheDocument();
});

test("DEPLOYMENT_FAILURE", () => {
  render(<StatusIcon status={DeploymentStatus.DEPLOYMENT_FAILURE} />, {});

  expect(screen.getByTestId("deployment-error-icon")).toBeInTheDocument();
});

test("DEPLOYMENT_PENDING", () => {
  render(<StatusIcon status={DeploymentStatus.DEPLOYMENT_PENDING} />, {});

  expect(screen.getByTestId("deployment-pending-icon")).toBeInTheDocument();
});

test("DEPLOYMENT_PLANNED", () => {
  render(<StatusIcon status={DeploymentStatus.DEPLOYMENT_PLANNED} />, {});

  expect(screen.getByTestId("deployment-pending-icon")).toBeInTheDocument();
});

test("DEPLOYMENT_ROLLING_BACK", () => {
  render(<StatusIcon status={DeploymentStatus.DEPLOYMENT_ROLLING_BACK} />, {});

  expect(screen.getByTestId("deployment-rollback-icon")).toBeInTheDocument();
});

test("DEPLOYMENT_RUNNING", () => {
  render(<StatusIcon status={DeploymentStatus.DEPLOYMENT_RUNNING} />, {});

  expect(screen.getByTestId("deployment-running-icon")).toBeInTheDocument();
});

test("DEPLOYMENT_SUCCESS", () => {
  render(<StatusIcon status={DeploymentStatus.DEPLOYMENT_SUCCESS} />, {});

  expect(screen.getByTestId("deployment-success-icon")).toBeInTheDocument();
});
