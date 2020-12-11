import { AnyAction, combineReducers } from "redux";
import { ThunkAction } from "redux-thunk";
import { activeStageSlice } from "./active-stage";
import { applicationsSlice } from "./applications";
import { applicationLiveStateSlice } from "./applications-live-state";
import { deploymentsSlice } from "./deployments";
import { environmentsSlice } from "./environments";
import { pipedsSlice } from "./pipeds";
import { stageLogsSlice } from "./stage-logs";
import { toastsSlice } from "./toasts";
import { commandsSlice } from "./commands";
import { applicationFilterOptionsSlice } from "./application-filter-options";
import { meSlice } from "./me";
import { deploymentFilterOptionsSlice } from "./deployment-filter-options";
import { loginSlice } from "./login";
import { projectSlice } from "./project";
import { deploymentConfigsSlice } from "./deployment-configs";
import { sealedSecretSlice } from "./sealed-secret";
import { apiKeysSlice } from "./api-keys";
import { updateApplicationSlice } from "./update-application";

export const reducers = combineReducers({
  deployments: deploymentsSlice.reducer,
  deploymentFilterOptions: deploymentFilterOptionsSlice.reducer,
  applicationLiveState: applicationLiveStateSlice.reducer,
  applications: applicationsSlice.reducer,
  applicationFilterOptions: applicationFilterOptionsSlice.reducer,
  updateApplication: updateApplicationSlice.reducer,
  stageLogs: stageLogsSlice.reducer,
  activeStage: activeStageSlice.reducer,
  pipeds: pipedsSlice.reducer,
  environments: environmentsSlice.reducer,
  commands: commandsSlice.reducer,
  toasts: toastsSlice.reducer,
  me: meSlice.reducer,
  login: loginSlice.reducer,
  project: projectSlice.reducer,
  deploymentConfigs: deploymentConfigsSlice.reducer,
  sealedSecret: sealedSecretSlice.reducer,
  apiKeys: apiKeysSlice.reducer,
});

export type AppState = ReturnType<typeof reducers>;
export type AppThunk = ThunkAction<
  Promise<unknown>,
  AppState,
  unknown,
  AnyAction
>;
