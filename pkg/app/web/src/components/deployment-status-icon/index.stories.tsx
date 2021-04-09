import * as React from "react";
import { DeploymentStatusIcon } from "./";
import { DeploymentStatus } from "pipe/pkg/app/web/model/deployment_pb";

export default {
  title: "DEPLOYMENT/StatusIcon",
  component: DeploymentStatusIcon,
};

export const overview: React.FC = () => (
  <>
    <DeploymentStatusIcon status={DeploymentStatus.DEPLOYMENT_SUCCESS} />
    <DeploymentStatusIcon status={DeploymentStatus.DEPLOYMENT_FAILURE} />
    <DeploymentStatusIcon status={DeploymentStatus.DEPLOYMENT_CANCELLED} />
    <DeploymentStatusIcon status={DeploymentStatus.DEPLOYMENT_PENDING} />
    <DeploymentStatusIcon status={DeploymentStatus.DEPLOYMENT_PLANNED} />
    <DeploymentStatusIcon status={DeploymentStatus.DEPLOYMENT_RUNNING} />
  </>
);
