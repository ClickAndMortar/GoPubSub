const webpack = require("webpack");
const path = require("path");

let config = {
    entry: "./assets/main.js",
    output: {
        path: path.resolve(__dirname, "./static"),
        filename: "./bundle.js"
    },
    module: {
        rules: [
            {
                test: /\.css$/,
                loader: ['style-loader', 'css-loader']
            }
        ]
    }
};

module.exports = config;
