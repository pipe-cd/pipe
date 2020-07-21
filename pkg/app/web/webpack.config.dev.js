/* eslint @typescript-eslint/no-var-requires: 0 */
"use strict";
require("dotenv").config();
const path = require("path");
const webpack = require("webpack");
const ForkTsCheckerWebpackPlugin = require("fork-ts-checker-webpack-plugin");
const mergeConfig = require("webpack-merge");
const webpackBaseConfig = require("./webpack.common");

module.exports = (env, argv) =>
  mergeConfig(webpackBaseConfig(env, argv), {
    mode: process.env.NODE_ENV === "production" ? "production" : "development",
    devtool: "inline-source-map",
    entry: {
      index: "./src/index.tsx",
    },
    resolve: {
      extensions: [".mjs", ".ts", ".tsx", ".js"],
      alias: {
        pipe: path.resolve(__dirname, "../../../bazel-bin/"),
      },
      modules: [path.resolve(__dirname, "node_modules"), "node_modules"],
    },
    devServer: {
      contentBase: path.join(__dirname, "dist"),
      compress: true,
      port: 9090,
      historyApiFallback: true,
      disableHostCheck: true,
      proxy: {
        "/api": {
          changeOrigin: true,
          target: process.env.API_ENDPOINT,
          pathRewrite: { "^/api": "" },
          withCredentials: true,
          headers: {
            Cookie: process.env.API_COOKIE,
          },
        },
      },
    },
    module: {
      rules: [
        {
          test: /\.tsx?$/,
          loader: "ts-loader",
          options: {
            transpileOnly: true,
          },
        },
        {
          type: "javascript/auto",
          test: /\.mjs$/,
          use: [],
        },
      ],
    },
    plugins: [
      new ForkTsCheckerWebpackPlugin(),
      new webpack.EnvironmentPlugin(["API_ENDPOINT"]),
    ],
  });
