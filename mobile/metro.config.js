const { getDefaultConfig } = require('expo/metro-config');

// This app lives in an npm workspace; the shared @newsletter/sdk package
// (sdk/typescript) is linked through the workspace root's node_modules.
// Expo's metro config detects the workspace root automatically.
const config = getDefaultConfig(__dirname);

module.exports = config;
