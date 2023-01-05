let webpack = require('webpack');

module.exports = {
  plugins: [
    new webpack.ContextReplacementPlugin(/moment[\/\\]locale$/, /en/)
  ],
  resolve: {
      fallback: {
          "fs": false
      }
  }

};
