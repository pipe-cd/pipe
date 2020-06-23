import { makeStyles } from "@material-ui/core";
import React, { FC, memo, useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useParams } from "react-router";
import { DeploymentDetail } from "../../components/deployment-detail";
import { LogViewer } from "../../components/log-viewer";
import { Pipeline } from "../../components/pipeline";
import { AppState } from "../../modules";
import {
  Deployment,
  DeploymentStatus,
  fetchDeploymentById,
  selectById as selectDeploymentById,
} from "../../modules/deployments";
import { useInterval } from "../../utils/use-interval";

const FETCH_INTERVAL = 4000;

const useStyles = makeStyles({
  root: {
    display: "flex",
    flexDirection: "column",
    alignItems: "stretch",
    flex: 1,
  },
  main: {
    flex: 1,
  },
  bottomContent: {
    flex: "initial",
  },
  loading: {
    flex: 1,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  },
});

function isRunningDeployment(status: DeploymentStatus | undefined): boolean {
  if (status === undefined) {
    return false;
  }

  return [
    DeploymentStatus.DEPLOYMENT_PENDING,
    DeploymentStatus.DEPLOYMENT_PLANNED,
    DeploymentStatus.DEPLOYMENT_ROLLING_BACK,
    DeploymentStatus.DEPLOYMENT_RUNNING,
  ].includes(status);
}

export const DeploymentDetailPage: FC = memo(function DeploymentDetailPage() {
  const classes = useStyles();
  const dispatch = useDispatch();
  const { deploymentId } = useParams<{ deploymentId: string }>();
  const deployment = useSelector<AppState, Deployment | undefined>((state) =>
    selectDeploymentById(state.deployments, deploymentId)
  );

  useEffect(() => {
    if (deploymentId) {
      dispatch(fetchDeploymentById(deploymentId));
    }
  }, [dispatch, deploymentId]);

  useInterval(
    () => {
      dispatch(fetchDeploymentById(deploymentId));
    },
    deploymentId && isRunningDeployment(deployment?.status)
      ? FETCH_INTERVAL
      : null
  );

  return (
    <div className={classes.root}>
      <div className={classes.main}>
        <DeploymentDetail deploymentId={deploymentId} />
        <Pipeline deploymentId={deploymentId} />
      </div>
      <div className={classes.bottomContent}>
        <LogViewer />
      </div>
    </div>
  );
});
