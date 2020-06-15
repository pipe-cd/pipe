import { AnyAction, combineReducers } from "redux";
import { ThunkAction, ThunkDispatch } from "redux-thunk";
import { deploymentsSlice } from "./deployments";
import { applicationsSlice } from "./applications";
import { stageLogsSlice } from "./stage-logs";
import { activeStageSlice } from "./active-stage";

export const reducers = combineReducers({
  deployments: deploymentsSlice.reducer,
  applications: applicationsSlice.reducer,
  stageLogs: stageLogsSlice.reducer,
  activeStage: activeStageSlice.reducer,
});

export type AppState = ReturnType<typeof reducers>;
export type AppThunk = ThunkAction<
  Promise<unknown>,
  AppState,
  unknown,
  AnyAction
>;
export type AppDispatch = ThunkDispatch<AppState, null, AnyAction>;
