// Custom webpack config that doesn't use .config
const path = require('path');
module.exports = {
  entry: './src/index.ts',
  output: {
    path: path.resolve(__dirname, 'dist'),
  }
};
