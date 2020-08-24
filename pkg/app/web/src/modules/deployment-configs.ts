import { createSlice, createAsyncThunk } from "@reduxjs/toolkit";
import * as deploymentConfigAPI from "../api/deployment-config";
import {
  DeploymentConfigTemplateLabel,
  DeploymentConfigTemplate,
} from "pipe/pkg/app/web/api_client/service_pb";
import { addApplication } from "./applications";

interface DeploymentConfigs {
  templates: Record<string, DeploymentConfigTemplate.AsObject[]>;
  targetApplicationId: string | null;
}
export type DeploymentConfigTemplateLabelKey = keyof typeof DeploymentConfigTemplateLabel;
const initialState: DeploymentConfigs = {
  templates: {},
  targetApplicationId: null,
};

export const fetchTemplateList = createAsyncThunk<
  DeploymentConfigTemplate.AsObject[],
  { applicationId: string; labels: DeploymentConfigTemplateLabel[] }
>("deploymentConfigs/fetchTemplates", async ({ labels, applicationId }) => {
  const {
    templatesList,
  } = await deploymentConfigAPI.getDeploymentConfigTemplates({
    applicationId,
    labelsList: labels,
  });
  return templatesList;
});

export const deploymentConfigsSlice = createSlice({
  name: "deploymentConfigs",
  initialState,
  reducers: {
    clearTemplateTarget(state) {
      state.targetApplicationId = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchTemplateList.pending, (state) => {})
      .addCase(fetchTemplateList.fulfilled, (state, action) => {
        state.templates[action.meta.arg.applicationId] = action.payload;
      })
      .addCase(fetchTemplateList.rejected, (state, action) => {
        console.error(action);
      })
      .addCase(addApplication.fulfilled, (state, action) => {
        state.targetApplicationId = action.payload;
      });
  },
});

export const selectTemplateByAppId = (
  state: DeploymentConfigs
): DeploymentConfigTemplate.AsObject | null => {
  if (!state.targetApplicationId) {
    return null;
  }

  const templates = state.templates[state.targetApplicationId];

  if (templates === undefined) {
    return null;
  }

  return templates[0] || null;
};

export const { clearTemplateTarget } = deploymentConfigsSlice.actions;

export {
  DeploymentConfigTemplateLabel,
  DeploymentConfigTemplate,
} from "pipe/pkg/app/web/api_client/service_pb";
