exports.handler = async (event, context, callback) => {
  const { config, request } = event.Records[0].cf

  return callback(null, {
    headers: {
      ...request.headers,
      "x-served-by": [
        {
          key: "x-served-by",
          value: "edwardofclt/cloudfront-lambda@edge-emulator",
        },
      ],
    },
  })
}
