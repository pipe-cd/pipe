module.exports = {
  roots: ["<rootDir>/src"],
  transform: {
    "^.+\\.tsx?$": "ts-jest",
    "\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga|ico)$":
      "<rootDir>/file-transformer.js",
  },
  moduleNameMapper: {
    "^pipe/(.*)$": "<rootDir>/../../../$1",
  },
  moduleDirectories: ["node_modules", "__fixtures__"],
  coveragePathIgnorePatterns: [
    "/node_modules/",
    ".test.ts",
    ".stories.ts",
    ".d.ts",
  ],
  clearMocks: true,
  setupFiles: ["./jest.setup.js"],
  setupFilesAfterEnv: ["./jest.after-env.ts"],
  coverageReporters: ["lcovonly", "text-summary"],
  globals: {
    "ts-jest": {
      diagnostics: {
        warnOnly: true,
      },
    },
  },
};
