import { Story } from "@storybook/react";
import { dummyApplication } from "~/__fixtures__/dummy-application";
import { dummyEnv } from "~/__fixtures__/dummy-environment";
import { createDecoratorRedux } from "~~/.storybook/redux-decorator";
import { ApplicationList, ApplicationListProps } from ".";

export default {
  title: "APPLICATION/ApplicationList",
  component: ApplicationList,
  argTypes: {
    onPageChange: {
      actions: "onPageChange",
    },
  },
  decorators: [
    createDecoratorRedux({
      applications: {
        adding: false,
        disabling: {},
        syncing: {},
        entities: {
          [dummyApplication.id]: dummyApplication,
        },
        ids: [dummyApplication.id],
      },
      environments: {
        entities: {
          [dummyEnv.id]: dummyEnv,
        },
        ids: [dummyEnv.id],
      },
    }),
  ],
};

const Template: Story<ApplicationListProps> = (args) => (
  <ApplicationList {...args} />
);

export const Overview = Template.bind({});
Overview.args = {
  currentPage: 1,
};
