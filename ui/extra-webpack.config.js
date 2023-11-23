let webpack = require('webpack');
let path = require("path");
let MONACO_DIR = path.join(__dirname, "node_modules/monaco-editor");

module.exports = {
  plugins: [
    new webpack.ContextReplacementPlugin(/moment[\/\\]locale$/, /en/)
  ],
  module: {
    rules: [
      {
        test: /\.css$/,
        include: MONACO_DIR,
        use: ["style-loader", {
         "loader": "css-loader",
         "options": {
           "url": false,
         },
        }],
      },
    ],
  },
  resolve: {
      fallback: {
          "fs": false
      }
  }

};
