import { action } from "@storybook/addon-actions";
import React from "react";
import { TextWithCopyButton } from "./text-with-copy-button";

export default {
  title: "TextWithCopyButton",
  component: TextWithCopyButton,
};

export const overview: React.FC = () => (
  <TextWithCopyButton
    value="hello"
    onCopy={action("onCopy")}
    label="Copy text"
  />
);
